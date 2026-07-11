package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Config represents settings saved in config.json
type Config struct {
	Provider      string   `json:"provider"`
	Model         string   `json:"model"`
	APIKey        string   `json:"api_key"`
	Categories    []string `json:"categories"`
	CustomBaseURL string   `json:"custom_base_url"` // New field
}

// FileInfo represents file metadata sent to frontend
type FileInfo struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	Extension      string `json:"extension"`
	SizeBytes      int64  `json:"size_bytes"`
	SnippetPreview string `json:"snippet_preview"`
	IsImage        bool   `json:"is_image"`
}

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

const ConfigFile = "config.json"
var DefaultCategories = []string{"Invoices", "Receipts", "Readings", "Code", "Images", "Others"}
var secretKey = []byte("sortmindai-secret-encryptionkey!") // 32 bytes for AES-256

// encrypt encrypts plaintext using AES-256-CFB
func encrypt(text string) (string, error) {
	if text == "" {
		return "", nil
	}
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}
	ciphertext := make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(text))
	return hex.EncodeToString(ciphertext), nil
}

// decrypt decrypts ciphertext using AES-256-CFB
func decrypt(cryptoText string) (string, error) {
	if cryptoText == "" {
		return "", nil
	}
	ciphertext, err := hex.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return string(ciphertext), nil
}

// loadConfig reads the config file or returns defaults
func (a *App) loadConfig() Config {
	cfg := Config{
		Provider:      "gemini",
		Model:         "gemini-2.5-flash",
		APIKey:        "",
		Categories:    DefaultCategories,
		CustomBaseURL: "",
	}

	if _, err := os.Stat(ConfigFile); err == nil {
		data, err := os.ReadFile(ConfigFile)
		if err == nil {
			var loaded Config
			if err := json.Unmarshal(data, &loaded); err == nil {
				if loaded.Provider != "" {
					cfg.Provider = loaded.Provider
				}
				if loaded.Model != "" {
					cfg.Model = loaded.Model
				}
				if loaded.APIKey != "" {
					decrypted, err := decrypt(loaded.APIKey)
					if err == nil {
						cfg.APIKey = decrypted
					} else {
						// Fallback: key might be plaintext (migration)
						cfg.APIKey = loaded.APIKey
					}
				}
				if len(loaded.Categories) > 0 {
					cfg.Categories = loaded.Categories
				}
				cfg.CustomBaseURL = loaded.CustomBaseURL
			}
		}
	}
	return cfg
}

// saveConfig writes configuration to disk (encrypting the API key)
func (a *App) saveConfig(cfg Config) error {
	encryptedKey, err := encrypt(cfg.APIKey)
	if err == nil {
		cfg.APIKey = encryptedKey
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, data, 0644)
}

// GetSettings loads settings for frontend
func (a *App) GetSettings() Config {
	return a.loadConfig()
}

// SaveSettings saves user settings from frontend
func (a *App) SaveSettings(cfg Config) (string, error) {
	err := a.saveConfig(cfg)
	if err != nil {
		return "", err
	}
	return "Settings saved successfully.", nil
}

// SelectFolder opens a native OS directory dialog
func (a *App) SelectFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Folder to Organize",
	})
}

// ScanFolder scans the directory and returns list of files with preview snippets
func (a *App) ScanFolder(folderPath string) ([]FileInfo, error) {
	folderPath = strings.TrimSpace(folderPath)
	if folderPath == "" {
		return nil, fmt.Errorf("folder path is empty")
	}

	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".csv": true, ".json": true, ".xml": true, ".yaml": true, ".yml": true,
		".html": true, ".css": true, ".js": true, ".ts": true, ".py": true, ".java": true, ".cpp": true,
		".c": true, ".sh": true, ".bat": true, ".ini": true, ".cfg": true, ".log": true, ".sql": true,
		".go": true, ".rs": true,
	}
	imageExtensions := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".bmp": true,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		filePath := filepath.Join(folderPath, entry.Name())
		ext := strings.ToLower(filepath.Ext(entry.Name()))

		fileItem := FileInfo{
			Name:      entry.Name(),
			Path:      filePath,
			Extension: ext,
			SizeBytes: info.Size(),
			IsImage:   imageExtensions[ext],
		}

		// Read snippet or metadata
		if textExtensions[ext] {
			fileItem.SnippetPreview = readTextSnippet(filePath, 1000)
		} else if fileItem.IsImage {
			fileItem.SnippetPreview = fmt.Sprintf("Image file. Size: %s", formatBytes(info.Size()))
		} else if ext == ".pdf" {
			fileItem.SnippetPreview = readPdfRawSnippet(filePath, 1000)
		} else {
			fileItem.SnippetPreview = fmt.Sprintf("Binary file (%s). Size: %s", ext, formatBytes(info.Size()))
		}

		files = append(files, fileItem)
	}

	return files, nil
}

func readTextSnippet(path string, maxChars int) string {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Sprintf("[Read error: %s]", err)
	}
	defer f.Close()

	buf := make([]byte, maxChars)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Sprintf("[Read error: %s]", err)
	}
	return string(buf[:n])
}

// readPdfRawSnippet extracts metadata or printable streams from PDF headers safely
func readPdfRawSnippet(path string, maxChars int) string {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Sprintf("[Read PDF error: %s]", err)
	}
	defer f.Close()

	buf := make([]byte, maxChars)
	n, _ := f.Read(buf)
	
	// Clean PDF buffer into printable characters
	var printable strings.Builder
	count := 0
	for i := 0; i < n; i++ {
		c := buf[i]
		if (c >= 32 && c <= 126) || c == '\n' || c == '\r' || c == '\t' {
			printable.WriteByte(c)
			count++
		}
		if count >= 800 {
			break
		}
	}
	res := printable.String()
	if len(res) == 0 {
		return "PDF File (Binary header)"
	}
	return "PDF Content Stream: " + res
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGT"[exp])
}

// ---------------------------------------------------------------------------
// HTTP API Integrations
// ---------------------------------------------------------------------------

func callGeminiAPI(apiKey, model, prompt string, base64Image string, mimeType string) (string, error) {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	var requestBody map[string]interface{}

	if base64Image != "" {
		// Multimodal content
		requestBody = map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{"text": prompt},
						{
							"inlineData": map[string]string{
								"mimeType": mimeType,
								"data":     base64Image,
							},
						},
					},
				},
			},
		}
	} else {
		// Text-only content
		requestBody = map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]interface{}{
						{"text": prompt},
					},
				},
			},
		}
	}

	jsonBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API Error (Status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse Response
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text), nil
	}

	return "", fmt.Errorf("no response text candidate received from Gemini")
}

func callOpenAICompatibleAPI(customURL, apiKey, model, prompt string, base64Image string, mimeType string) (string, error) {
	if model == "" {
		model = "gpt-4o-mini"
	}
	url := "https://api.openai.com/v1/chat/completions"
	if customURL != "" {
		url = customURL
	}

	var content []interface{}
	content = append(content, map[string]interface{}{
		"type": "text",
		"text": prompt,
	})

	if base64Image != "" {
		content = append(content, map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]string{
				"url": fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image),
			},
		})
	}

	requestBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": content,
			},
		},
		"temperature": 0.0,
		"max_tokens":  20,
	}

	jsonBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API Error (Status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	if len(result.Choices) > 0 {
		return strings.TrimSpace(result.Choices[0].Message.Content), nil
	}

	return "", fmt.Errorf("no choice text received from OpenAI-compatible endpoint")
}

func callClaudeAPI(apiKey, model, prompt string) (string, error) {
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	url := "https://api.anthropic.com/v1/messages"

	requestBody := map[string]interface{}{
		"model":      model,
		"max_tokens": 20,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	jsonBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Claude API Error (Status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
			Type string `json:"type"`
		} `json:"content"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	if len(result.Content) > 0 {
		return strings.TrimSpace(result.Content[0].Text), nil
	}

	return "", fmt.Errorf("no response text received from Claude")
}

func callOllamaAPI(model, prompt string, base64Image string) (string, error) {
	if model == "" {
		model = "llama3"
	}
	url := "http://localhost:11434/api/generate"

	requestBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.0,
		},
	}

	if base64Image != "" {
		requestBody["images"] = []string{base64Image}
	}

	jsonBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama connection failed (Status %d)", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	return strings.TrimSpace(result.Response), nil
}

// TestConnection tests the configured AI settings
func (a *App) TestConnection(provider, model, apiKey, customBaseURL string) (string, error) {
	testPrompt := "Reply with only the word OK."
	var result string
	var err error

	switch provider {
	case "gemini":
		if apiKey == "" {
			return "", fmt.Errorf("API Key is empty")
		}
		result, err = callGeminiAPI(apiKey, model, testPrompt, "", "")
	case "openai":
		if apiKey == "" {
			return "", fmt.Errorf("API Key is empty")
		}
		result, err = callOpenAICompatibleAPI("", apiKey, model, testPrompt, "", "")
	case "deepseek":
		if apiKey == "" {
			return "", fmt.Errorf("API Key is empty")
		}
		url := "https://api.deepseek.com/v1/chat/completions"
		if customBaseURL != "" {
			url = customBaseURL
		}
		result, err = callOpenAICompatibleAPI(url, apiKey, model, testPrompt, "", "")
	case "anthropic":
		if apiKey == "" {
			return "", fmt.Errorf("API Key is empty")
		}
		result, err = callClaudeAPI(apiKey, model, testPrompt)
	case "custom":
		if customBaseURL == "" {
			return "", fmt.Errorf("Custom Base URL is required for custom provider")
		}
		result, err = callOpenAICompatibleAPI(customBaseURL, apiKey, model, testPrompt, "", "")
	case "ollama":
		result, err = callOllamaAPI(model, testPrompt, "")
	default:
		return "", fmt.Errorf("unknown provider name: %s", provider)
	}

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Connected successfully! Response: %s", result), nil
}

// ---------------------------------------------------------------------------
// File Classification & Move Operations
// ---------------------------------------------------------------------------

// OrganizeFile handles single-file classification and moving
func (a *App) OrganizeFile(filePath string, execute bool) (map[string]interface{}, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist")
	}

	cfg := a.loadConfig()
	
	// 1. Gather file details
	filename := filepath.Base(filePath)
	ext := strings.ToLower(filepath.Ext(filename))
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	fileInfoItem := FileInfo{
		Name:      filename,
		Path:      filePath,
		Extension: ext,
		SizeBytes: info.Size(),
	}

	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".csv": true, ".json": true, ".xml": true, ".yaml": true, ".yml": true,
		".html": true, ".css": true, ".js": true, ".ts": true, ".py": true, ".java": true, ".cpp": true,
		".c": true, ".sh": true, ".bat": true, ".ini": true, ".cfg": true, ".log": true, ".sql": true,
		".go": true, ".rs": true,
	}

	if textExtensions[ext] {
		fileInfoItem.SnippetPreview = readTextSnippet(filePath, 1000)
	} else if ext == ".pdf" {
		fileInfoItem.SnippetPreview = readPdfRawSnippet(filePath, 1000)
	} else {
		fileInfoItem.SnippetPreview = fmt.Sprintf("Binary file. Size: %d bytes.", info.Size())
	}

	// Read image data as base64 if multimodal-compatible
	base64Image := ""
	mimeType := ""
	imageExtensions := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true,
	}
	if imageExtensions[ext] {
		fileInfoItem.IsImage = true
		imgData, err := os.ReadFile(filePath)
		if err == nil {
			base64Image = base64.StdEncoding.EncodeToString(imgData)
			mimeType = "image/jpeg"
			if ext == ".png" {
				mimeType = "image/png"
			} else if ext == ".webp" {
				mimeType = "image/webp"
			} else if ext == ".gif" {
				mimeType = "image/gif"
			}
		}
	}

	// 2. Build Prompt
	categoriesList := strings.Join(cfg.Categories, ", ")
	prompt := fmt.Sprintf(`You are a professional file organizer helper. Categorize the file below into EXACTLY ONE folder from this list: [%s].

Instructions:
1. Respond with ONLY the exact folder name from the list.
2. Do NOT write any code, markdown, reasoning, explanation, or punctuation.
3. If none of the categories fit the file contents well, return 'Others'.
4. Do NOT make up new folder names. Choose only from the list.

File Details:
- Name: %s
- Extension: %s
- Size: %d bytes
- Content/Snippet Preview:
---
%s
---

Response (ONLY the category name):`, categoriesList, fileInfoItem.Name, fileInfoItem.Extension, fileInfoItem.SizeBytes, fileInfoItem.SnippetPreview)

	// 3. Call AI
	var category string
	var apiErr error

	switch cfg.Provider {
	case "gemini":
		category, apiErr = callGeminiAPI(cfg.APIKey, cfg.Model, prompt, base64Image, mimeType)
	case "openai":
		category, apiErr = callOpenAICompatibleAPI("", cfg.APIKey, cfg.Model, prompt, base64Image, mimeType)
	case "deepseek":
		url := "https://api.deepseek.com/v1/chat/completions"
		if cfg.CustomBaseURL != "" {
			url = cfg.CustomBaseURL
		}
		category, apiErr = callOpenAICompatibleAPI(url, cfg.APIKey, cfg.Model, prompt, base64Image, mimeType)
	case "anthropic":
		category, apiErr = callClaudeAPI(cfg.APIKey, cfg.Model, prompt)
	case "custom":
		url := cfg.CustomBaseURL
		if url == "" {
			url = "https://api.openai.com/v1/chat/completions"
		}
		category, apiErr = callOpenAICompatibleAPI(url, cfg.APIKey, cfg.Model, prompt, base64Image, mimeType)
	case "ollama":
		category, apiErr = callOllamaAPI(cfg.Model, prompt, base64Image)
	default:
		apiErr = fmt.Errorf("invalid provider config: %s", cfg.Provider)
	}

	if apiErr != nil {
		return nil, apiErr
	}

	// Clean category response
	cleaned := strings.TrimSpace(category)
	cleaned = strings.Trim(cleaned, `'"*` + "`" + `#._`)
	
	matchedCategory := "Others"
	for _, cat := range cfg.Categories {
		if strings.ToLower(cleaned) == strings.ToLower(cat) {
			matchedCategory = cat
			break
		}
	}

	movedTo := ""
	if execute {
		// Move file
		baseDir := filepath.Dir(filePath)
		destDir := filepath.Join(baseDir, matchedCategory)
		os.MkdirAll(destDir, 0755)

		destPath := filepath.Join(destDir, filename)
		
		// Handle collisions
		counter := 1
		nameWithoutExt := filename[:len(filename)-len(ext)]
		for {
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				break
			}
			destPath = filepath.Join(destDir, fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext))
			counter++
		}

		err = os.Rename(filePath, destPath)
		if err != nil {
			// Fallback helper in case os.Rename fails across filesystem bounds
			err = moveCopyFallback(filePath, destPath)
		}
		
		if err != nil {
			return nil, fmt.Errorf("failed to move file: %s", err)
		}
		movedTo = destPath
	}

	return map[string]interface{}{
		"success":  true,
		"filename": filename,
		"category": matchedCategory,
		"moved_to": movedTo,
	}, nil
}

// moveCopyFallback copies and deletes if Rename fails across device boundaries
func moveCopyFallback(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	in.Close() // Close early before removing
	return os.Remove(src)
}

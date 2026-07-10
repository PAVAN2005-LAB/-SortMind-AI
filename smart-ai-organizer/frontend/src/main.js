import './style.css';
import './app.css';

// Import Wails runtime and bindings
import {
    GetSettings,
    SaveSettings,
    TestConnection,
    SelectFolder,
    ScanFolder,
    OrganizeFile
} from '../wailsjs/go/main/App';

// ── Model catalog per provider ──
const MODELS = {
    gemini: [
        { value: "gemini-2.5-flash",  label: "Gemini 2.5 Flash (Recommended)" },
        { value: "gemini-2.5-pro",    label: "Gemini 2.5 Pro" },
        { value: "gemini-2.0-flash",  label: "Gemini 2.0 Flash" },
        { value: "gemini-1.5-flash",  label: "Gemini 1.5 Flash" },
        { value: "gemini-1.5-pro",    label: "Gemini 1.5 Pro" },
    ],
    openai: [
        { value: "gpt-4o-mini", label: "GPT-4o Mini (Fast & Cheap)" },
        { value: "gpt-4o",      label: "GPT-4o" },
    ],
    ollama: [
        { value: "llama3",  label: "Llama 3" },
        { value: "mistral", label: "Mistral" },
        { value: "phi3",    label: "Phi 3" },
        { value: "gemma2",  label: "Gemma 2" },
    ],
};

// ── State ──
let config = { provider: "gemini", model: "gemini-2.5-flash", api_key: "", categories: [] };
let files  = [];
let isPaused = false;
let isCancelled = false;

// ── DOM refs ──
const $ = id => document.getElementById(id);

const selProvider   = $("selProvider");
const selModel      = $("selModel");
const inpApiKey     = $("inpApiKey");
const btnToggleKey  = $("btnToggleKey");
const apiKeyBlock   = $("apiKeyBlock");
const btnTest       = $("btnTest");
const btnSave       = $("btnSave");
const categoryChips = $("categoryChips");
const inpNewCat     = $("inpNewCat");
const btnAddCat     = $("btnAddCat");
const connBadge     = $("connectionBadge");
const connLabel     = $("connectionLabel");
const connDot       = connBadge.querySelector(".dot");

const inpFolder     = $("inpFolder");
const btnBrowse     = $("btnBrowse");
const btnDemo       = $("btnDemo");
const btnScan       = $("btnScan");

const filesPanel    = $("filesPanel");
const spanCount     = $("spanCount");
const spanSel       = $("spanSel");
const tBody         = $("tBody");
const chkHead       = $("chkHead");
const btnSelAll     = $("btnSelAll");
const btnSelNone    = $("btnSelNone");

const btnDry        = $("btnDry");
const btnExec       = $("btnExec");
const btnPause      = $("btnPause");
const btnCancel     = $("btnCancel");

const logCard       = $("logCard");
const terminal      = $("terminal");
const btnClearLog   = $("btnClearLog");

// ── Init ──
document.addEventListener("DOMContentLoaded", async () => {
    wireEvents();
    await loadSettings();
});

function wireEvents() {
    selProvider.addEventListener("change", () => {
        fillModels(selProvider.value);
        apiKeyBlock.style.display = selProvider.value === "ollama" ? "none" : "block";
    });
    btnToggleKey.addEventListener("click", () => {
        const show = inpApiKey.type === "password";
        inpApiKey.type = show ? "text" : "password";
        btnToggleKey.textContent = show ? "🔒" : "👁️";
    });
    btnTest.addEventListener("click", testConn);
    btnSave.addEventListener("click", saveSettings);
    btnAddCat.addEventListener("click", addCat);
    inpNewCat.addEventListener("keydown", e => { if (e.key === "Enter") addCat(); });

    btnBrowse.addEventListener("click", browseFolder);
    btnDemo.addEventListener("click", () => { inpFolder.value = "demo_cluttered_folder"; log("Demo path loaded.", "sys"); });
    btnScan.addEventListener("click", scanDir);

    chkHead.addEventListener("change", () => setAllChecks(chkHead.checked));
    btnSelAll.addEventListener("click",  () => setAllChecks(true));
    btnSelNone.addEventListener("click", () => setAllChecks(false));

    btnDry.addEventListener("click",  () => runOrganize(false));
    btnExec.addEventListener("click", () => runOrganize(true));
    btnClearLog.addEventListener("click", () => { terminal.innerHTML = '<div class="tline sys">[system] Log cleared.</div>'; });

    // Pause/Cancel listeners
    btnPause.addEventListener("click", () => {
        isPaused = !isPaused;
        btnPause.textContent = isPaused ? "Resume" : "Pause";
        log(isPaused ? "Execution paused." : "Execution resumed.", "warn");
    });
    btnCancel.addEventListener("click", () => {
        isCancelled = true;
        btnCancel.textContent = "Cancelling…";
        btnCancel.disabled = true;
        log("Cancel requested. Stopping run after the current file is processed…", "warn");
    });
}

// ── Models dropdown ──
function fillModels(provider, selected) {
    selModel.innerHTML = "";
    (MODELS[provider] || []).forEach(m => {
        const o = document.createElement("option");
        o.value = m.value;
        o.textContent = m.label;
        if (m.value === selected) o.selected = true;
        selModel.appendChild(o);
    });
}

// ── Settings ──
async function loadSettings() {
    try {
        config = await GetSettings();
    } catch (e) {
        log("Failed to load settings: " + e, "err");
    }

    selProvider.value = config.provider || "gemini";
    fillModels(config.provider || "gemini", config.model);
    inpApiKey.value = config.api_key || "";
    apiKeyBlock.style.display = config.provider === "ollama" ? "none" : "block";
    renderChips();
    log("Settings loaded from backend.", "sys");
}

async function saveSettings() {
    config.provider = selProvider.value;
    config.model    = selModel.value;
    config.api_key  = inpApiKey.value.trim();

    btnSave.textContent = "Saving…"; btnSave.disabled = true;
    try {
        const msg = await SaveSettings(config);
        log(msg, "ok");
    } catch (err) {
        log("Save error: " + err, "err");
    }
    btnSave.textContent = "Save Settings"; btnSave.disabled = false;
}

// ── Browse Folder ──
async function browseFolder() {
    try {
        const path = await SelectFolder();
        if (path) {
            inpFolder.value = path;
            log(`Selected directory: "${path}"`, "sys");
        }
    } catch (err) {
        log("Error picking folder: " + err, "err");
    }
}

// ── Connection ──
async function testConn() {
    const provider = selProvider.value;
    const model    = selModel.value;
    const apiKey   = inpApiKey.value.trim();

    setConn("yellow", "Testing…");
    log(`Testing connection to ${provider} / ${model}…`, "act");
    btnTest.textContent = "Testing…"; btnTest.disabled = true;

    try {
        const msg = await TestConnection(provider, model, apiKey);
        setConn("green", provider.toUpperCase() + " Ready");
        log("Connection OK: " + msg, "ok");
    } catch (err) {
        setConn("red", "Failed");
        log("Connection failed: " + err, "err");
    }

    btnTest.textContent = "Test Connection"; btnTest.disabled = false;
}

function setConn(color, text) {
    connDot.className = "dot " + color;
    connLabel.textContent = text;
}

// ── Categories ──
function renderChips() {
    categoryChips.innerHTML = "";
    (config.categories || []).forEach(c => {
        const d = document.createElement("div"); d.className = "chip";
        d.innerHTML = `<span>${c}</span>` + (c !== "Others" ? `<button class="x" data-cat="${c}">✕</button>` : "");
        categoryChips.appendChild(d);
    });
    categoryChips.querySelectorAll(".x").forEach(b =>
        b.addEventListener("click", () => { removeCat(b.dataset.cat); })
    );
}

function addCat() {
    let v = inpNewCat.value.trim();
    if (!v) return;
    v = v.charAt(0).toUpperCase() + v.slice(1);
    if (config.categories.map(c=>c.toLowerCase()).includes(v.toLowerCase())) return;
    config.categories.push(v);
    renderChips(); inpNewCat.value = "";
    log(`Category "${v}" added. Click Save Settings to persist.`, "sys");
}

function removeCat(name) {
    config.categories = config.categories.filter(c => c !== name);
    renderChips();
    log(`Category "${name}" removed. Click Save Settings to persist.`, "sys");
}

// ── Scan ──
async function scanDir() {
    const folder = inpFolder.value.trim();
    if (!folder) { alert("Please select or enter a folder path."); return; }

    log(`Scanning "${folder}"…`, "act");
    btnScan.textContent = "Scanning…"; btnScan.disabled = true;

    try {
        files = await ScanFolder(folder);
        renderTable();
        filesPanel.style.display = files.length ? "flex" : "none";
        logCard.style.display = "flex";
        log(`Found ${files.length} file(s).`, "ok");
    } catch (err) {
        log("Scan error: " + err, "err");
        alert("Scan failed: " + err);
    }

    btnScan.textContent = "Scan Directory"; btnScan.disabled = false;
}

// ── Table helpers ──
function icon(ext) {
    ext = ext.toLowerCase();
    if ([".py",".js",".ts",".css",".html",".java",".cpp",".c",".sh",".bat",".go",".rs"].includes(ext)) return "⚙️";
    if ([".txt",".md",".log",".ini",".cfg"].includes(ext)) return "📄";
    if (ext === ".pdf") return "📕";
    if ([".png",".jpg",".jpeg",".webp",".gif",".bmp"].includes(ext)) return "🖼️";
    if ([".csv",".json",".xml",".yaml",".yml"].includes(ext)) return "📊";
    return "📦";
}

function fmtSize(b) {
    if (!b) return "0 B";
    const u = ["B","KB","MB","GB"];
    const i = Math.floor(Math.log(b)/Math.log(1024));
    return (b/Math.pow(1024,i)).toFixed(i?1:0) + " " + u[i];
}

function renderTable() {
    tBody.innerHTML = "";
    if (files.length === 0) {
        tBody.innerHTML = `<tr><td colspan="6" style="text-align: center; color: var(--text-dim); padding: 24px;">No files found.</td></tr>`;
        return;
    }
    files.forEach((f, i) => {
        const tr = document.createElement("tr"); tr.id = "row-" + i;
        tr.innerHTML = `
            <td class="col-chk"><input type="checkbox" class="fchk" data-i="${i}" checked></td>
            <td><div class="fname"><span class="ficon">${icon(f.extension)}</span>${f.name}</div></td>
            <td><div class="fsnippet" title="${esc(f.snippet_preview)}">${esc(f.snippet_preview) || "—"}</div></td>
            <td class="col-size">${fmtSize(f.size_bytes)}</td>
            <td class="col-cat" id="cat-${i}">—</td>
            <td class="col-status"><span class="tag pending" id="tag-${i}">Pending</span></td>
        `;
        tBody.appendChild(tr);
    });
    tBody.querySelectorAll(".fchk").forEach(c => c.addEventListener("change", updateSelCount));
    spanCount.textContent = files.length;
    chkHead.checked = true;
    updateSelCount();
}

function esc(s) { return (s||"").replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;"); }

function setAllChecks(v) {
    tBody.querySelectorAll(".fchk").forEach(c => c.checked = v);
    chkHead.checked = v;
    updateSelCount();
}

function updateSelCount() {
    const n = tBody.querySelectorAll(".fchk:checked").length;
    spanSel.textContent = n;
    chkHead.checked = n === files.length && files.length > 0;
}

// Helper to pause/cancel checking inside the loop
async function checkPauseCancel() {
    while (isPaused && !isCancelled) {
        await new Promise(resolve => setTimeout(resolve, 200));
    }
    if (isCancelled) {
        throw new Error("CancelledByUser");
    }
}

// ── Organize ──
async function runOrganize(execute) {
    const checked = [...tBody.querySelectorAll(".fchk:checked")];
    if (!checked.length) { alert("Select at least one file to organize."); return; }
    if (execute && !confirm(`Move ${checked.length} file(s) into category folders?`)) return;

    // Reset control states
    isPaused = false;
    isCancelled = false;
    btnPause.textContent = "Pause";
    btnCancel.disabled = false;
    btnCancel.textContent = "Cancel";

    // Show control buttons
    btnPause.style.display = "inline-flex";
    btnCancel.style.display = "inline-flex";

    lockUI(true);
    const mode = execute ? "EXECUTE" : "DRY RUN";
    log(`Starting ${mode} for ${checked.length} files…`, "act");

    let ok = 0;
    for (const cb of checked) {
        // Check pause/cancel state before processing
        try {
            await checkPauseCancel();
        } catch (err) {
            if (err.message === "CancelledByUser") {
                log("Process cancelled by user.", "err");
                break;
            }
        }

        const i = +cb.dataset.i;
        const f = files[i];
        const row = $("row-" + i);
        const tag = $("tag-" + i);
        const cat = $("cat-" + i);

        row.classList.add("working");
        tag.className = "tag working"; tag.textContent = "Sorting…";

        try {
            const res = await OrganizeFile(f.path, execute);
            row.classList.remove("working");

            if (res.success) {
                ok++;
                cat.textContent = res.category;
                cat.style.color = "var(--primary)"; cat.style.fontWeight = "600";
                if (execute) {
                    tag.className = "tag moved"; tag.textContent = "Moved";
                    log(`✔ "${f.name}" → ${res.category}/`, "ok");
                    cb.checked = false;
                } else {
                    tag.className = "tag predicted"; tag.textContent = "Preview";
                    log(`⤷ "${f.name}" → ${res.category}`, "warn");
                }
            } else {
                tag.className = "tag error"; tag.textContent = "Error";
                log(`✗ "${f.name}": Unknown error`, "err");
            }
        } catch (err) {
            row.classList.remove("working");
            tag.className = "tag error"; tag.textContent = "Error";
            log(`✗ "${f.name}": ${err}`, "err");
        }
        updateSelCount();
    }

    // Hide control buttons
    btnPause.style.display = "none";
    btnCancel.style.display = "none";

    lockUI(false);
    log(`${mode} complete — ${ok}/${checked.length} succeeded.`, "ok");

    if (execute && ok > 0 && !isCancelled) {
        log("Refreshing folder…", "sys");
        await scanDir();
    }
}

function lockUI(on) {
    [btnDry, btnExec, btnScan, btnSave, btnTest, btnAddCat, btnBrowse].forEach(b => b.disabled = on);
    tBody.querySelectorAll(".fchk").forEach(c => c.disabled = on);
    chkHead.disabled = on;
}

// ── Logger ──
function log(msg, cls = "") {
    const d = document.createElement("div");
    d.className = "tline " + cls;
    const t = new Date().toLocaleTimeString();
    d.textContent = `[${t}] ${msg}`;
    terminal.appendChild(d);
    terminal.scrollTop = terminal.scrollHeight;
    logCard.style.display = "flex";
}

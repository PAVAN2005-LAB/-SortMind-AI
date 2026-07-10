import os
import shutil
from PIL import Image, ImageDraw

def create_demo_files():
    folder_name = "demo_cluttered_folder"
    if os.path.exists(folder_name):
        shutil.rmtree(folder_name)
    os.makedirs(folder_name)

    print(f"Creating demo cluttered folder at: {os.path.abspath(folder_name)}")

    # 1. Starbucks Receipt
    with open(os.path.join(folder_name, "receipt_starbucks_102.txt"), "w", encoding="utf-8") as f:
        f.write("""STARBUCKS COFFEE #10394
Seattle, WA 98101
Date: 2026-06-15 08:34 AM

1x Caffe Latte (Grande)   $4.75
1x Blueberry Scone         $3.25

Subtotal:                  $8.00
Tax (10%):                 $0.80
Total:                     $8.80

Paid: Visa ****1234
Thank you for your visit!
""")

    # 2. Cloud Invoice
    with open(os.path.join(folder_name, "invoice_cloud_hosting_9921.txt"), "w", encoding="utf-8") as f:
        f.write("""INVOICE - AWS Cloud Services
Invoice #: INV-2026-9921
Billing Period: June 2026
Date of Issue: July 1, 2026
Due Date: July 31, 2026

Bill To:
Pavan
123 Tech Street, Dev City

Itemized Charges:
1. EC2 Instance - t3.medium (720 hrs)   $36.00
2. S3 Storage - Standard 500GB          $11.50
3. RDS Postgres Database                $95.00

Total Amount Due:                       $142.50
Please pay via bank transfer to account ending in 9876.
""")

    # 3. ML Paper/Reading
    with open(os.path.join(folder_name, "deep_learning_paper_intro.txt"), "w", encoding="utf-8") as f:
        f.write("""Title: Deep Learning for Computer Vision in Autonomous Driving
Abstract:
This paper presents a comprehensive review of deep convolutional neural networks (CNNs)
and transformer architectures applied to real-time object detection and semantic segmentation.
We evaluate performance across various datasets such as COCO and Cityscapes, and discuss 
the trade-offs between accuracy and latency in edge computing devices. Furthermore, we 
introduce a novel spatial attention mechanism that improves lane detection by 14% while 
maintaining a inference rate of 60 frames per second on standard embedded hardware.
Keywords: Deep Learning, Computer Vision, Autonomous Vehicles, Transformers.
""")

    # 4. Code file (Python)
    with open(os.path.join(folder_name, "factorial_calculation.py"), "w", encoding="utf-8") as f:
        f.write("""def factorial(n):
    \"\"\"
    Calculate the factorial of a non-negative integer n.
    \"\"\"
    if n == 0 or n == 1:
        return 1
    return n * factorial(n - 1)

if __name__ == "__main__":
    number = 5
    result = factorial(number)
    print(f"The factorial of {number} is {result}")
""")

    # 5. Code file (CSS)
    with open(os.path.join(folder_name, "index_style.css"), "w", encoding="utf-8") as f:
        f.write("""body {
    background-color: #0d1117;
    color: #c9d1d9;
    font-family: 'Outfit', sans-serif;
    margin: 0;
    padding: 0;
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100vh;
}
.card {
    background: rgba(255, 255, 255, 0.05);
    backdrop-filter: blur(10px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    padding: 24px;
    box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.37);
}
""")

    # 6. An Image file (visual placeholder using PIL)
    img = Image.new("RGB", (300, 200), color=(30, 41, 59))
    draw = ImageDraw.Draw(img)
    # Draw simple shapes to simulate a photo / graph
    draw.rectangle([50, 50, 250, 150], fill=(71, 85, 105), outline=(148, 163, 184), width=3)
    draw.ellipse([100, 70, 200, 130], fill=(59, 130, 246))
    img.save(os.path.join(folder_name, "vacation_chart_preview.png"))

    # 7. Another Image file (Receipt style image)
    receipt_img = Image.new("RGB", (300, 400), color=(245, 245, 245))
    draw_receipt = ImageDraw.Draw(receipt_img)
    # Draw horizontal lines simulating text lines in a receipt
    draw_receipt.text((20, 20), "OFFICE DEPOT", fill=(0, 0, 0))
    draw_receipt.line([20, 50, 280, 50], fill=(120, 120, 120), width=2)
    draw_receipt.text((20, 70), "Paper Reams  x4  $24.00", fill=(0, 0, 0))
    draw_receipt.text((20, 100), "Ink Cartridges x2 $85.00", fill=(0, 0, 0))
    draw_receipt.text((20, 150), "TOTAL: $109.00", fill=(0, 0, 0))
    receipt_img.save(os.path.join(folder_name, "office_depot_receipt.jpg"))

    print("Demo cluttered folder successfully generated with 7 diverse files!")

if __name__ == "__main__":
    create_demo_files()

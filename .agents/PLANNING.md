# PLANNING.md: High-Performance Go AWB Generator

## 1. Project Overview
The goal is to replace a slow WeasyPrint-based PDF generation system with a high-performance Go CLI binary. The tool will accept AWB (Air Waybill) data in JSON format and output a single, merged, print-ready PDF.

**Performance Targets:**
*   **Speed:** < 2 seconds for 1,000 labels.
*   **Memory:** < 50MB peak RSS for 5,000 labels.
*   **Output:** Vector-based PDF (no raster images for barcodes).

## 2. Tech Stack
*   **Language:** Go (Latest stable)
*   **PDF Engine:** `github.com/johnfercher/maroto/v2` (Grid-based layout)
*   **Barcode Engine:** `github.com/boombuler/barcode`
*   **CLI Framework:** `github.com/spf13/cobra`
*   **Concurrency:** Standard Goroutines + Sync Groups

## 3. Data Schema (Input JSON)
The binary must accept a JSON array of AWB objects:
```json
[
  {
    "awb_number": "ZFW123456789",
    "order_id": "#9982",
    "sender": "Store Name",
    "receiver": "John Doe",
    "address": "123 Main St, Mumbai, India",
    "pincode": "400001",
    "weight": "0.5kg",
    "sku_details": "Item A x 1, Item B x 2"
  }
]
```

## 4. Execution Phases

### Phase 1: Core Layout & Static Prototype
- [ ] Initialize Go module `awb-gen`.
- [ ] Set up `go:embed` to bundle TrueType Fonts (.ttf) into the binary.
- [ ] Create a `generator` package to define the label layout (150mm x 100mm).
- [ ] Implement a function to draw a single label with:
    - Text blocks for Sender/Receiver.
    - A Code128 Barcode (Vector-drawn).
    - Boarder lines and SKU table.
- [ ] **Output:** Successfully generate a `test.pdf` with one static label.

### Phase 2: Parallel Generation Engine
- [ ] Implement a Worker Pool pattern.
- [ ] Create a `Processor` that:
    - Splits the input JSON into chunks.
    - Assigns chunks to $N$ workers (where $N = $ CPU cores).
    - Generates PDF pages in memory.
- [ ] Implement PDF Merging: Efficiently combine individual pages into one final document.

### Phase 3: CLI & Python Bridge
- [ ] Integrate `Cobra` for the CLI interface.
- [ ] Define flags: `--input` (JSON string or file path) and `--output` (target PDF path).
- [ ] Implement `stdin` support so Python can pipe JSON directly:
    `cat data.json | ./awb-gen --stdin --output batch.pdf`
- [ ] Ensure non-zero exit codes on failure and structured error logging.

### Phase 4: Optimization & QA
- [ ] **Benchmarking:** Create a script to generate 5,000 dummy AWBs and measure execution time.
- [ ] **Visual Regression:** Ensure barcodes are crisp and scannable at various zoom levels.
- [ ] **Memory Profiling:** Use `pprof` to ensure no memory leaks during large batch processing.

## 5. Implementation Guidelines (For the Agent)
1.  **Direct Drawing:** Do not use any "HTML to PDF" libraries. Use direct PDF primitives (Rectangles, Lines, Text).
2.  **Vector Barcodes:** Convert the barcode data into Maroto/GoPDF primitive shapes. This ensures high-speed printing and small file sizes.
3.  **No External Dependencies:** The final binary must be statically linked and contain all fonts. It should run on a fresh Linux/Ubuntu server without installing `libpangocairo` or other C-libs.
4.  **Error Handling:** If a single AWB in a batch of 1,000 fails (e.g., data corruption), log the error but attempt to continue with the others, marking the failed one in the logs.

## 6. Integration Snippet (For Python/Django)
The agent should provide this snippet to help integrate the tool:
```python
import subprocess
import json

def generate_awbs_go(awb_data, output_path):
    json_payload = json.dumps(awb_data).encode('utf-8')
    process = subprocess.Popen(
        ['./awb-gen', '--output', output_path, '--stdin'],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE
    )
    stdout, stderr = process.communicate(input=json_payload)
    if process.returncode != 0:
        raise Exception(f"Go Generator Error: {stderr.decode()}")
    return output_path
```

## 7. Success Criteria
1.  **Binary Portability:** A single `awb-gen` file that runs on Linux.
2.  **Performance:** 1,000 labels in < 1.5 seconds on a standard 2-core VPS.
3.  **Reliability:** Zero "Segmentation Faults" or "Out of Memory" errors during a 10,000-label stress test.

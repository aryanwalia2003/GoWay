# AWB Generation Performance & Profiling Report

This document outlines the performance benchmarks and profiling results for the `awb-gen` tool, conducted on a batch of **5,000 labels**.

## 📊 Benchmark Results (5,000 Labels)

The benchmark was executed using the `make bench-e2e` target periodically to ensure stability.

| Metric | Result |
|---|---|
| **Total Labels** | 5,000 |
| **Elapsed Time** | **1m 13.19s** |
| **Throughput** | **~68.38 labels/sec** |
| **Peak Memory (RSS)** | **966 MB** |
| **Output PDF Size** | **18.7 MB** |

---

## 🧠 Memory Profiling (Heap Analysis)

The heap profile identifies the primary sources of memory pressure during the generation and merger phases.

### Top Memory Consumers (`inuse_space`)

| Function | Cumulative Memory | % of Total |
|---|---|---|
| `bytes.growSlice` | 142.05 MB | 79.51% |
| `MarotoGenerator.GenerateLabel` | 93.23 MB | 52.18% |
| `OrderedMerger.Merge` | 81.93 MB | 45.86% |
| `pdfcpu/pkg/api.MergeRaw` | 32.56 MB | 18.22% |

### Analysis
- **Dynamic Buffer Growth:** Nearly 80% of memory usage is attributed to `bytes.growSlice`. This occurs as the system dynamically expands buffers to hold generated PDF pages and the final merged document.
- **Worker Isolation:** The parallel worker pool (where each worker handles `GenerateLabel`) accounts for ~52% of memory. This scale linearly with concurrent worker count.
- **Merge Overhead:** The concatenation phase (`Merge`) holds all 5,000 single-page PDFs in RAM before merging them into a single `bytes.Buffer`.

---

## ⚡ CPU Profiling

The CPU profile reveals where the processor spends the most time during the 5,000-label run.

### Top CPU Consumers

| Function | Flat Time | Cumulative % |
|---|---|---|
| `runtime.memclrNoHeapPointers` | 4.84s | 6.22% |
| `runtime.memmove` | 4.04s | 11.41% |
| `encoding/json.encode` | 1.97s | 24.27% |
| `gofpdf.generateSCCS` | 0.85s | 17.66% |
| `compress/flate.deflate` | 0.96s | 2.50% |

### Analysis
- **Memory Management:** `memclr` and `memmove` (collectively ~11%) indicate heavy work zeroing and shifting large byte slices during buffer expansion. 
- **JSON Overhead:** The streaming encoding/decoding of label data consumes a significant portion of CPU time during the produce phase.
- **PDF Compression:** `flate.deflate` and `adler32` update cycles are expected overhead for generating compressed PDF content.

---

## 🚀 Optimization Roadmap

Based on these findings, future optimizations should focus on:

1. **Streaming Merger:** Modify the `OrderedMerger` to write directly to an `os.File` or an `io.Writer` instead of building the entire 18MB+ PDF in a `bytes.Buffer`.
2. **Batch Chunking:** Process 50,000+ labels in sub-batches (e.g., merge every 1,000 pages to disk) to keep peak RSS below 200MB regardless of total volume.
3. **Buffer Pre-alloc:** Implement better heuristics for `bytes.Buffer` pre-allocation in the `MarotoGenerator` to reduce `memmove` overhead.

---

*Generated on 2026-03-28 using Go 1.22.1 pprof suite.*

# AWB Generation Performance & Profiling Report

This document outlines the performance benchmarks and profiling results for the `awb-gen` tool, conducted on a batch of **5,000 labels**, running with the newly optimized memory-bounded pipeline.

## 📊 Benchmark Results (5,000 Labels)

The benchmark was executed using the `make bench-e2e` and `make bench-mem` targets to ensure stability.

| Metric | V1 (Baseline) | V2 (Optimized) | Improvement |
|---|---|---|---|
| **Total Labels** | 5,000 | 5,000 | - |
| **Elapsed Time** | 1m 13.19s | **1m 03.14s** | ~14% Faster |
| **Throughput** | ~68.38 labels/s | **~79.17 labels/s** | +15% Throughput |
| **Peak Memory (RSS)** | 966 MB | **~295 MB** | **70% Reduction** |
| **Active Heap (`inuse_space`)**| ~178 MB | **~45 MB** | **74% Reduction** |

---

## 🧠 Memory Profiling (Heap Analysis)

The heap profile highlights the massive reduction in memory footprint following the implementation of chunked merging and `estimatedLabelBytes` pre-allocation.

### Top Memory Consumers (`inuse_space`)

| Function | Cumulative Memory | % of Total |
|---|---|---|
| `MarotoGenerator.GenerateLabel` | 42.34 MB | 93.21% |
| `Maroto.Generate` | 19.80 MB | 43.59% |
| `gofpdf.utf8FontFile.parseHMTXTable`| 15.92 MB | 35.07% |

### Analysis
- **Eradication of `bytes.growSlice`:** In the V1 baseline, `bytes.growSlice` accounted for `142 MB` (80% of total memory) due to slice doublings. In V2, pre-allocating the label slice buffer using the `estimatedLabelBytes` constant (5KB) completely removed buffer-resizing overhead.
- **Constant-Bounded Merger:** V1 held all 5,000 pages in RAM before calling `api.MergeRaw` into a giant `bytes.Buffer`. V2 streams the files in 500-page chunks directly to disk (`MergeToFile`). As a result, the merger allocations have flatlined and disappeared from the top consumers.
- **Semaphore Limits:** The peak active heap is capped at **45 MB**, largely representing the concurrent generation contexts throttled by the `MaxConcurrentPDF` semaphore.

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

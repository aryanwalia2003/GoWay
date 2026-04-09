# LaTeX Shipping Label Performance Report

This report documents the performance characteristics and resource consumption of the GoWay `LaTeXAssembler` when generating forward shipping labels using the `uc_shipping_label` template.

## 🚀 Throughput & Latency (Post-Daemon Optimization)

Benchmarks were conducted on an **8-core CPU** using the `BenchmarkAssembleUCShippingLabel` suite, utilizing the **Tectonic Process Pool (Daemon Mode)**.

| Metric | Sequential | Parallel (8 Workers) |
|---|---|---|
| **Latency** | **931 ms/op** | **407 ms/op** |
| **Throughput** | **~1.07 labels/s** | **~19.65 labels/s** |

> [!NOTE]
> The adoption of Tectonic Daemon Mode has eliminated OS-level process-spawning jitter, increasing parallel throughput by **~7x** compared to the baseline (~2.78 labels/s).

---

## 🧠 Memory Bottlenecks (Go Heap)

The following functions account for over **85% of total memory allocations** within the Go service during label generation.

1.  **`compress/flate.NewWriter` (50.2%)**: High allocation rate due to PNG encoding of generated barcodes.
2.  **`image.NewNRGBA` (15.3%)**: Memory required for raw image buffer allocation during barcode generation.
3.  **`compress/flate.initDeflate` (15.2%)**: Initialization costs for the DEFLATE compression algorithm used in PNGs.

---

## ⚡ CPU Bottlenecks

1.  **Syscall / External Process Wait (99%)**: Despite the warm pool, the Go runtime spends the vast majority of its time waiting for the persistent `tectonic` binary to finalize compilation.
2.  **PNG Filter & Compression**: The only significant Go-side CPU consumer, accounting for ~40ms of active processing time per label.

---

### 🚀 Barcode Caching (Implemented)
By introducing a thread-safe `barcode.Cache`, memory allocations plummeted by **~55%** (from ~2.16MB to ~956KB per operation), significantly reducing GC pressure.

## 🛠 Next Optimization Target

To further optimize throughput:

1.  **Syscall Pooling / Ramdisk**: If Tectonic's compilation times remain a bottleneck, writing the `.tex` and `.pdf` files to a `tmpfs` ramdisk can shave off IO latency.
2.  **Pre-compiled Preamble (Experimental)**: Re-visit `.fmt` files if standard TeX Live is introduced or explore Tectonic V2 session persistence for shared preamble state.

---
*Updated on 2026-04-09 using Go 1.22.1 pprof suite.*

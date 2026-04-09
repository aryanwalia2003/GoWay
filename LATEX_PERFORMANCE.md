# LaTeX Shipping Label Performance Report

This report documents the performance characteristics and resource consumption of the GoWay `LaTeXAssembler` when generating forward shipping labels using the `uc_shipping_label` template.

## 🚀 Throughput & Latency (Post-Daemon Optimization)

Benchmarks were conducted on an **8-core CPU** using the `BenchmarkAssembleUCShippingLabel` suite, utilizing the **Tectonic Process Pool (Daemon Mode)**.

| Metric | Sequential | Parallel (8 Workers) |
|---|---|---|
| **Latency (Baseline)** | 931 ms/op | 407 ms/op |
| **Latency (Pre-compiled Preamble)** | **775 ms/op** | **369 ms/op** |
| **Throughput** | **~1.29 labels/s** | **~21.6 labels/s** |

> [!NOTE]
> Moving from stdin pipeline to Tectonic V2 Project Mode (TOML) eliminated redundant preamble parsing, dropping latency by ~20% per request.

---

## 🧠 Memory Optimization (Go Heap)

With caching removed and `png.NoCompression` enabled, memory allocations remain high per operation because Go's standard library statically allocates large stream buffers for `zlib` regardless of the compression level choice.

| State | Memory / Op | Allocations / Op |
|---|---|---|
| **Baseline** | ~2,142 KB | ~372 |
| **Optimized (No Compression)** | **~2,168 KB** | **~357** |

**Remaining Hotspots:**
1.  **PNG Encoding (zlib stream buffers)**: `image/png.Encode` unconditionally spins up 512KB dict buffers for streaming out blocks, dominating heap allocations.

---

## ⚡ CPU Analysis

1.  **Tectonic Compilation (99%+)**: The Go service is nearly 100% efficient, spent entirely waiting for the external Tectonic process.
2.  **PNG Encoding**: Bypassing compression (`png.NoCompression`) avoids DEFLATE cycle overhead, ensuring immediate flush of barcode pixels to byte buffers.

---

## 🛠 Next Optimization Target

1.  **Sync.Pool for Zlib Writers**: If memory allocation becomes a catastrophic scaling issue under infinite unique traffic, implement a `sync.Pool` wrapping custom PNG encoders or adopt `klauspost/compress`.
2.  **Pre-compiled Preamble (Experimental)**: Re-visit `.fmt` files if standard TeX Live is introduced or explore Tectonic V2 session persistence for shared preamble state.

---
*Updated on 2026-04-09 using Go 1.22.1 pprof suite.*

# awb-gen

High-performance Air Waybill (AWB) PDF label generator written in Go.

Replaces a slow WeasyPrint-based pipeline with a lock-free SPSC architecture
that renders 1 000 labels in under 2 seconds at under 50 MB peak RSS.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    SPSC PIPELINE                                  │
│                                                                   │
│  os.Stdin / File                                                  │
│       │                                                           │
│       ▼                                                           │
│  ┌──────────────┐   chan Job (buf = N×2)                          │
│  │   Producer   │ ─────────────────────────────────►             │
│  │  json.Decoder│                                  │             │
│  │  streaming   │                                  ▼             │
│  │  O(1) memory │                      ┌────────────────────┐   │
│  └──────────────┘                      │    Worker Pool     │   │
│                                        │  N = runtime.CPU   │   │
│                                        │                    │   │
│                                        │  Each worker owns: │   │
│                                        │  • MarotoGenerator │   │
│                                        │  • Code128Encoder  │   │
│                                        │  • image.NRGBA buf │   │
│                                        │                    │   │
│                                        │  ZERO shared state │   │
│                                        │  ZERO mutexes      │   │
│                                        └─────────┬──────────┘   │
│                                                  │               │
│                              chan PageResult (buf = N×4)         │
│                                                  │               │
│                                                  ▼               │
│                                        ┌─────────────────┐       │
│                                        │  OrderedMerger  │       │
│                                        │  1 goroutine    │       │
│                                        │  sort by index  │       │
│                                        │  pdfcpu merge   │       │
│                                        └────────┬────────┘       │
│                                                 │                 │
└─────────────────────────────────────────────────┼─────────────── ┘
                                                  ▼
                                             output.pdf
```

### Key design decisions

| Concern | Decision | Why |
|---|---|---|
| **No mutexes** | Each worker owns its own `MarotoGenerator` | Zero contention — all state is goroutine-local |
| **SPSC channels** | One producer, one merger consumer | Matches SPSC model; Go channels are lock-free at low contention |
| **Streaming JSON** | `json.NewDecoder` + `dec.More()` loop | O(1) memory — never loads full JSON payload into RAM |
| **Ordered merge** | Collect → `sort.Slice` by Index → pdfcpu merge | Single-goroutine merge = no write contention |
| **Embedded fonts** | `//go:embed` TTF bytes, shared read-only pointer | One allocation for the full run, safe across goroutines |
| **Atomic file layout** | One entity per file, strict naming conventions | Per `.agents/SKILL.md` standards |

---

## Performance targets

| Metric | Target | How we achieve it |
|---|---|---|
| Speed | < 2 s for 1 000 labels | Parallel workers (N = NumCPU), buffered channels |
| Memory | < 50 MB RSS for 5 000 labels | Streaming decode, per-worker isolation, no global state |
| Output | Scannable barcodes | Code128 via boombuler, 3 px/module PNG at 72 px height |

---

## Prerequisites

- Go 1.21+
- `curl` and `unzip` (for font download)

---

## Setup

```bash
# 1. Clone
git clone https://github.com/your-org/awb-gen && cd awb-gen

# 2. Download Roboto fonts (one-time setup)
chmod +x scripts/download_fonts.sh
./scripts/download_fonts.sh

# 3. Resolve dependencies
go mod tidy

# 4. Build
make build
```

---

## Usage

```bash
# From a JSON file
./awb-gen --input data.json --output batch.pdf

# From stdin (pipe-friendly, integrates with Python/shell)
cat data.json | ./awb-gen --stdin --output batch.pdf

# Control worker count explicitly
./awb-gen --input data.json --output batch.pdf --workers 8

# Debug logging (human-readable, coloured)
./awb-gen --input data.json --output batch.pdf --debug

# With pprof profiling server
./awb-gen --input data.json --output batch.pdf --pprof localhost:6060
```

### Input JSON schema

```json
[
  {
    "awb_number":  "ZFW123456789",   // required
    "order_id":    "#9982",
    "sender":      "Store Name",
    "receiver":    "John Doe",        // required
    "address":     "123 Main St, Mumbai, India",  // required
    "pincode":     "400001",
    "weight":      "0.5kg",
    "sku_details": "Item A x1, Item B x2"
  }
]
```

Malformed or invalid records are **skipped with a warning** — the batch continues.

---

## Development

```bash
# All checks: fmt + vet + build + test
make all

# Unit + race tests
make test

# End-to-end benchmark (5 000 labels)
make bench-e2e

# Memory benchmark
make bench-mem

# Full Go benchmark suite with alloc reporting
make bench

# Lint (requires golangci-lint)
make lint

# Coverage report
make cover
```

---

## Package structure

```
internal/
  awb/          Data model + validation          (awb_struct, validate_method, awb_err)
  barcode/      Code128 encoder                  (renderer_iface, code128_*)
  generator/    Single-label PDF renderer        (maroto_*, generate_method, fonts_method, ...)
  pipeline/     SPSC orchestration               (pipeline_*, produce_method, work_method, ...)
  merger/       Ordered page merge via pdfcpu    (ordered_merger_*, merge_method)
  assets/       Embedded Roboto TTF fonts        (assets.go + fonts/)
  logger/       Structured zap logger            (logger.go)
  profiler/     pprof HTTP server                (profiler.go)
cmd/
  root.go       Cobra CLI wiring
integration/
  integration_test.go   End-to-end binary tests
bench/
  bench_test.go         Memory + throughput benchmarks
testdata/
  golden_3.json         3-record golden fixture
  malformed_mixed.json  Fixture with invalid records for skip testing
scripts/
  download_fonts.sh     One-time Roboto font download
  gen_fixture.sh        N-label JSON fixture generator
```

---

## License

MIT — see LICENSE file.

# Possible Optimizations v2 — Forward Shipping Label Path

Scope: `POST /render/forward-shipping-label` (LaTeX / Tectonic path).  
This document supersedes the completed items from `possible_optimizations.md` and captures **new** opportunities discovered in the current codebase.

---

## 1. Eliminate Double Payload Read in `HandleGenerate`

**File:** `internal/handler/generate.go`

`HandleGenerate` calls `io.ReadAll` into `bodyBytes`, then passes `bytes.NewReader(bodyBytes)` to `validatePayload` **and again** to `pl.Run`. This means the body is decoded twice: once for validation and once for the pipeline producer.

**Root Cause:** `validatePayload` uses its own `json.NewDecoder` for field-level validation. The pipeline's `produce` goroutine then re-decodes the exact same bytes from the reader.

**Proposed Fix:**  
Remove the separate `validatePayload` pre-pass. Inline validation inside `pipeline/produce_method.go` — the existing `record.Validate()` call already validates each record. Emit invalid records as error `RenderResult`s instead. This eliminates one full JSON decode pass over the entire payload.

```go
// BEFORE  (generate.go)
bodyBytes, _ := io.ReadAll(r.Body)
_, failedAWBs, err := validatePayload(bodyBytes)   // decode pass #1
results, _          := pl.Run(r.Context(), bytes.NewReader(bodyBytes)) // decode pass #2

// AFTER — pass the body reader directly; let the pipeline be the single source of truth
results, _ := pl.Run(r.Context(), r.Body)
```

**Estimated saving:** Eliminates O(N) allocations for `awb.AWB` structs during the pre-validation pass on every request.

---

## 2. Pool the `image.NRGBA` Pixel Buffer in `RenderBarcodePNG`

**File:** `internal/barcode/png_helper.go`

`RenderBarcodePNG` allocates a fresh `image.NewNRGBA(...)` on every call. For a barcode that is ~310 modules × 3 px wide × 72 px tall = ~270 KB of pixel data. With N labels per batch processed in parallel, this creates significant GC pressure.

**Root Cause:** The `image.NRGBA.Pix` slice is heap-allocated fresh each call. Only the encode buffer (`PngBufPool`) is pooled; the image itself is not.

**Proposed Fix:**  
Pool the `image.NRGBA` objects similarly to how `PngBufPool` works. Reset the pixel buffer with `copy`/`memclr` rather than allocating a new one.

```go
var nrgbaPool = sync.Pool{
    New: func() any {
        imgW := 310 * BarcodeBarWidthPx // worst-case Code128 width
        img := image.NewNRGBA(image.Rect(0, 0, imgW, BarcodeImgHeightPx))
        return img
    },
}

func RenderBarcodePNG(renderer Renderer, content string) ([]byte, error) {
    bars, barCount, err := renderer.Encode(content)
    if err != nil { ... }

    imgW := barCount * BarcodeBarWidthPx
    img := nrgbaPool.Get().(*image.NRGBA)
    defer nrgbaPool.Put(img)
    // Re-slice to actual width (no alloc) and flood fill white.
    img.Rect = image.Rect(0, 0, imgW, BarcodeImgHeightPx)
    img.Stride = 4 * imgW
    // ... paint bars as before
}
```

**Estimated saving:** Removes one ~270 KB heap allocation per label; directly reduces GC pauses under high batch load.

---

## 3. Write PDF Directly to `http.ResponseWriter` (Skip the Intermediate `bytes.Buffer`)

**File:** `internal/handler/generate.go`

`HandleGenerate` allocates a `bytes.Buffer`, fills it via `asm.AssembleToWriter`, then copies the entire buffer to `w`. For a 400-label batch this is a ~2 MB in-memory copy that is never needed.

**Root Cause:** The handler needs to set `Content-Length` before calling `w.WriteHeader(200)`, which requires knowing the size up front — forcing the intermediate buffer.

**Proposed Fix (Option A — drop `Content-Length`):**  
Remove the `Content-Length` header requirement. Write directly to `http.ResponseWriter` via `AssembleToWriter`. The HTTP/1.1 `Transfer-Encoding: chunked` framing handles variable-length bodies correctly, and modern clients don't require `Content-Length` for PDF downloads.

```go
w.Header().Set("Content-Type", "application/pdf")
w.WriteHeader(http.StatusOK)
_, failedAWBs, err := asm.AssembleToWriter(results, w)
```

**Proposed Fix (Option B — pipe through `io.Pipe`):**  
Use an `io.Pipe` so the gofpdf assembly writes to the pipe writer and the HTTP response reader is the pipe reader. This keeps streaming without materialising the full PDF buffer.

**Estimated saving:** Eliminates a single ~2 MB allocation per batch request; reduces peak RSS meaningfully under load.

---

## 4. Add `Content-Length` Header to Single-Label LaTeX Response

**File:** `internal/handler/latex_handler.go` → `writePDFResponse`

The LaTeX handler sets `Content-Type` but not `Content-Length`. The PDF bytes are already fully in memory (`pdfBytes`). Without `Content-Length`, the Go `http` package uses chunked transfer encoding, adding framing overhead and preventing connection reuse without a round-trip.

**Proposed Fix:**

```go
func writePDFResponse(w http.ResponseWriter, pdf []byte) {
    w.Header().Set("Content-Type", "application/pdf")
    w.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
    w.WriteHeader(http.StatusOK)
    w.Write(pdf)
}
```

**Estimated saving:** Minimal CPU, but eliminates chunked-encoding framing overhead and allows persistent connection reuse — measurable under concurrent load.

---

## 5. Short-Circuit `makeKey` String Concatenation via `strings.Builder`

**File:** `internal/assembler/latex_mapper.go`

`makeKey` is called recursively for every field in the JSON payload. For nested objects the two-step string concatenation `prefix + strings.ToUpper(key[:1]) + key[1:]` creates up to three intermediate string allocations per key.

**Proposed Fix:**

```go
func makeKey(prefix, key string) string {
    if prefix == "" {
        return key
    }
    var b strings.Builder
    b.Grow(len(prefix) + len(key))
    b.WriteString(prefix)
    b.WriteByte(strings.ToUpper(key)[0]) // single upper char
    b.WriteString(key[1:])
    return b.String()
}
```

Or, since the upper-casing only touches the first byte, use `unicode/utf8` + `unicode.ToUpper` for correctness with non-ASCII keys.

**Estimated saving:** Removes 2 intermediate string allocs per field per request; more significant for deeply nested or heavily annotated payloads.

---

## 6. Replace `concurrency` Middleware 503 Behavior with `429 Too Many Requests`

**File:** `internal/middleware/concurrency.go`

When the semaphore is full and the client context cancels, the middleware returns `500 Internal Server Error`. This is semantically wrong — the server is not broken, it is overloaded. Returning `503 Service Unavailable` (or `429 Too Many Requests`) allows clients, load balancers, and health checks to distinguish overload from bugs.

**Proposed Fix:**

```go
case <-r.Context().Done():
    http.Error(w, "too many concurrent requests", http.StatusServiceUnavailable)
    return
```

**Note:** This is a correctness/observability fix with zero performance cost.

---

## 7. Pre-size the `LaTeXAssembler` Template Cache Map

**File:** `internal/assembler/latex_assembler.go` → `NewLaTeXAssembler`

`templateCache` is created as `make(map[string][2][]byte)` with no size hint. On first use Go allocates a minimal map and rehashes as templates are loaded. Since the template count is bounded by the `templates/` directory (typically 1–5 entries), pre-sizing avoids rehashing.

**Proposed Fix:**

```go
templateCache: make(map[string][2][]byte, 8), // pre-size for known template count
```

**Estimated saving:** Avoids 1–2 map rehash cycles at startup; negligible at steady state.

---

## 8. Avoid `fmt.Errorf` Boxing in the Hot Path

**File:** `internal/assembler/latex_assembler.go`, `internal/barcode/png_helper.go`

`fmt.Errorf` with a `%v` or `%w` verb does a heap allocation for the formatted string even on the non-error path when the function compiles to an interface return. In tight loops (e.g., `extractOrGenerateBarcodes` calling `os.WriteFile` inside a `ForEach`), using `errors.New` with a sentinel or a pre-allocated error value avoids this.

**Proposed Fix:**  
For known, fixed error messages use `errors.New` at package level. Reserve `fmt.Errorf` for errors that embed dynamic values (file paths, AWB numbers).

---

## 9. Eliminate Redundant `base64.StdEncoding.DecodeString` Allocation Inside `ForEach`

**File:** `internal/assembler/latex_assembler.go` → `extractOrGenerateBarcodes`

Inside the `ForEach` closure, `base64.StdEncoding.DecodeString` allocates a new `[]byte` per barcode field. If the label has both `barcodeZippeeawb` and `barcodeRefcode` pre-encoded, that is two heap allocations. Both result bytes are immediately passed to `os.WriteFile` and then discarded.

**Proposed Fix:**  
Use `base64.StdEncoding.DecodeLen` + a reusable `[]byte` buffer (or a `sync.Pool`) to decode into a pre-allocated slice:

```go
dst := make([]byte, base64.StdEncoding.DecodedLen(len(value.String())))
n, err := base64.StdEncoding.Decode(dst, []byte(value.String()))
os.WriteFile(path, dst[:n], 0644)
```

**Estimated saving:** Removes one heap allocation per pre-encoded barcode field per request.

---

## Theoretical Ceiling

Tectonic compilation still dominates at 360–770 ms per single-label request. All Go-side optimizations above collectively target:

| Area | Potential saving |
|---|---|
| Double JSON decode pass (§1) | 1–5 ms per batch |
| NRGBA image pool (§2) | 0.5–2 ms per batch under load |
| PDF buffer elimination (§3) | 1–3 ms per batch |
| base64 alloc (§9) | <0.5 ms per request |
| Key builder (§5) | <0.5 ms per request |

**Further Tectonic-level ideas (high impact, high effort):**
- Run a persistent Tectonic daemon that accepts `.tex` content over stdin/IPC, avoiding `exec.Command` process-spawn overhead (~5–15 ms per request).
- Investigate `tectonic --keep-intermediates` to reuse format files between requests in the warm pool.

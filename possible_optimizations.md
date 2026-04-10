# Possible Optimizations for Forward Shipping Label Path

This document outlines precise optimizations for the `POST /render/forward-shipping-label` execution path in the GoWay shipping label microservice.

---

## 1. The JSON Round-Trip in the Handler

**File:** `internal/handler/latex_handler.go`

The request body arrives as JSON. Currently, it is decoded into a `map[string]interface{}` and then immediately re-encoded back to JSON bytes to pass to `Assemble`. This round-trip is unnecessary.

**Proposed Fix:**
Read the `data` field as `json.RawMessage` to capture raw bytes directly.

```go
type renderRequest struct {
    TemplateID string          `json:"template_id"`
    Data       json.RawMessage `json:"data"`
}

func (h *LaTeXHandler) render(...) {
    // req.Data is already the raw JSON bytes — pass directly
    pdfBytes, err := h.assembler.Assemble(r.Context(), req.TemplateID, req.Data)
    ...
}
```

---

## 2. Consolidate JSON Parsing

**File:** `internal/assembler/latex_assembler.go`

`executeWarmProcess` calls `MapToMacros` and `extractOrGenerateBarcodes` back-to-back, both of which perform `gjson.ParseBytes`.

**Proposed Fix:**
Parse once and pass the `gjson.Result` to both functions.

```go
// In executeWarmProcess:
parsed := gjson.ParseBytes(load)
macros = mapParsedToMacros(parsed)                
a.extractOrGenerateBarcodes(parsed, wp.srcDir)    
```

---

## 3. Global LaTeX Escaper

**File:** `internal/assembler/latex_mapper.go`

`strings.NewReplacer` is called on every field, causing multiple trie allocations per request.

**Proposed Fix:**
Use a package-level variable for the replacer.

```go
var latexEscaper = strings.NewReplacer(
    "%", "\\%", "$", "\\$", "&", "\\&", "#", "\\#",
    "_", "\\_", "{", "\\{", "}", "\\}",
)

func escapeLaTeX(val string) string {
    return latexEscaper.Replace(val)
}
```

---

## 4. Efficient Template Concatenation

**File:** `internal/assembler/latex_assembler.go`

Concatenating preamble, macros, and body using `+` creates multiple intermediate string allocations.

**Proposed Fix:**
Use `strings.Builder` with `Grow` to pre-allocate memory.

```go
var buf strings.Builder
buf.Grow(len(preambleContent) + 1 + len(macros) + 1 + len(bodyContent))
buf.Write(preambleContent)
buf.WriteByte('\n')
buf.WriteString(macros)
buf.WriteByte('\n')
buf.Write(bodyContent)

if err := os.WriteFile(indexFile, []byte(buf.String()), 0644); err != nil { ... }
```

---

## 5. Optimized Directory Cleanup

**File:** `internal/assembler/latex_assembler.go`

`resetWorkDir` uses `os.ReadDir` to find and remove `.png` files, which involves an expensive syscall to list the directory.

**Proposed Fix:**
Remove known files directly by name.

```go
func resetWorkDir(wp *warmProcess) error {
    toRemove := []string{
        filepath.Join(wp.srcDir, "index.tex"),
        filepath.Join(wp.srcDir, "_preamble.tex"),
        filepath.Join(wp.srcDir, "barcodeZippeeawb.png"),
        filepath.Join(wp.srcDir, "barcodeRefcode.png"),
    }
    for _, p := range toRemove {
        if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
            return err
        }
    }
    return nil
}
```

---

## 6. Stream PDF Output

**File:** `internal/assembler/latex_assembler.go` & `internal/handler/latex_handler.go`

The generated PDF is currently read into memory in its entirety before being written to the HTTP response.

**Proposed Fix:**
Stream the file directly from disk to the `io.Writer`.

```go
func (a *LaTeXAssembler) AssembleToWriter(ctx context.Context, templateID string, payload []byte, w io.Writer) error {
    // ... do everything the same, but at the end:
    f, err := os.Open(pdfPath)
    if err != nil {
        return err
    }
    defer f.Close()
    _, err = io.Copy(w, f)
    return err
}
```

---

## Theoretical Limits

Tectonic compilation accounts for 99%+ of CPU time (360ms - 770ms). These optimizations target the Go-side overhead, potentially saving 5-15ms per request. To increase throughput beyond this:
- **Calibrate Pool Size:** Tune `concurrencyLimit` to match CPU core count.
- **Tectonic Caching:** Ensure `~/.cache/Tectonic` persists and is on fast storage.

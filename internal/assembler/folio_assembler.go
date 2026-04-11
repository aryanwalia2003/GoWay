package assembler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sync"

	"awb-gen/internal/barcode"

	"github.com/carlos7ags/folio/document"
)

// templateEntry caches a pre-read HTML template string.
type templateEntry struct {
	html string
}

// FolioAssembler renders shipping-label PDFs from HTML templates.
// Static assets (logo, template HTML) are loaded once at construction time
// and reused on every Assemble call, eliminating per-request disk I/O.
type FolioAssembler struct {
	templateDir string
	BarcodeRend barcode.Renderer

	// cachedLogoURI is the base64 logo data URI, ready to embed in HTML.
	cachedLogoURI template.URL

	// templateCache stores pre-read HTML strings keyed by templateID.
	templateCacheMu sync.RWMutex
	templateCache   map[string]string
}

// NewFolioAssembler creates a FolioAssembler and eagerly loads the Zippee logo
// into memory as a base64-encoded data URI so the hot path is I/O-free.
func NewFolioAssembler(templateDir string) *FolioAssembler {
	a := &FolioAssembler{
		templateDir:   templateDir,
		BarcodeRend:   barcode.NewCode128Encoder(),
		templateCache: make(map[string]string, 4),
	}

	// Pre-load logo once at startup.
	logoPath := filepath.Join(templateDir, "zippee_logo_new.jpeg")
	if logoBytes, err := os.ReadFile(logoPath); err == nil {
		a.cachedLogoURI = template.URL("data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(logoBytes))
	}

	return a
}

// getTemplate returns the cached HTML string for templateID, loading and
// caching from disk on first access (lazy init for any extra templates).
func (a *FolioAssembler) getTemplate(templateID string) (string, error) {
	a.templateCacheMu.RLock()
	if html, ok := a.templateCache[templateID]; ok {
		a.templateCacheMu.RUnlock()
		return html, nil
	}
	a.templateCacheMu.RUnlock()

	// Not cached yet — load from disk and cache.
	tmplPath := filepath.Join(a.templateDir, templateID+".html")
	htmlBytes, err := os.ReadFile(tmplPath)
	if err != nil {
		return "", fmt.Errorf("failed to read HTML template %s: %w", tmplPath, err)
	}
	html := string(htmlBytes)

	a.templateCacheMu.Lock()
	a.templateCache[templateID] = html
	a.templateCacheMu.Unlock()

	return html, nil
}

func (a *FolioAssembler) Assemble(ctx context.Context, templateID string, payload []byte) ([]byte, error) {
	// 1. Parse JSON payload.
	var data map[string]any
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON payload: %v", err)
		}
	} else {
		data = make(map[string]any)
	}

	// 2. Inject cached logo — no disk I/O.
	if _, ok := data["zippeeLogo"]; !ok && a.cachedLogoURI != "" {
		data["zippeeLogo"] = a.cachedLogoURI
	}

	// 3. Generate barcodes if missing.
	if a.BarcodeRend != nil {
		if _, ok := data["barcodeZippeeawb"]; !ok {
			if val, ok := data["zippeeAwb"].(string); ok && val != "" {
				pngBytes, err := barcode.RenderBarcodePNG(a.BarcodeRend, val)
				if err == nil {
					data["barcodeZippeeawb"] = base64.StdEncoding.EncodeToString(pngBytes)
				}
			}
		}
		if _, ok := data["barcodeRefcode"]; !ok {
			if val, ok := data["referenceCode"].(string); ok && val != "" {
				pngBytes, err := barcode.RenderBarcodePNG(a.BarcodeRend, val)
				if err == nil {
					data["barcodeRefcode"] = base64.StdEncoding.EncodeToString(pngBytes)
				}
			}
		}
	}

	// 4. Get cached HTML template — disk read only on first call per templateID.
	htmlStr, err := a.getTemplate(templateID)
	if err != nil {
		return nil, err
	}

	// 5. Render via Folio.
	doc := document.NewDocument(document.PageSizeA4)
	if err := doc.AddHTMLTemplate(htmlStr, data, nil); err != nil {
		return nil, fmt.Errorf("failed to render HTML template to folio: %v", err)
	}

	return doc.ToBytes()
}

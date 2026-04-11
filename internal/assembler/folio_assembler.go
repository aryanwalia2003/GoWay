package assembler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"awb-gen/internal/barcode"

	"github.com/carlos7ags/folio/document"
)

type FolioAssembler struct {
	templateDir string
	BarcodeRend barcode.Renderer
}

func NewFolioAssembler(templateDir string) *FolioAssembler {
	return &FolioAssembler{
		templateDir: templateDir,
		BarcodeRend: barcode.NewCode128Encoder(),
	}
}

func (a *FolioAssembler) Assemble(ctx context.Context, templateID string, payload []byte) ([]byte, error) {
	// 1. Parse JSON payload
	var data map[string]any
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON payload: %v", err)
		}
	} else {
		data = make(map[string]any)
	}

	// 2. Inject logo as base64 data URI if not already set
	if _, ok := data["zippeeLogo"]; !ok {
		logoPath := filepath.Join(a.templateDir, "zippee_logo_new.jpeg")
		if logoBytes, err := os.ReadFile(logoPath); err == nil {
			// Use template.URL to mark the data URI as safe so html/template doesn't sanitize it
			data["zippeeLogo"] = template.URL("data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(logoBytes))
		}
	}

	// 3. Generate barcodes if missing
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

	// 3. Load HTML template string
	tmplPath := filepath.Join(a.templateDir, templateID+".html")
	htmlBytes, err := os.ReadFile(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTML template %s: %w", tmplPath, err)
	}

	// 4. Create Folio document and render template
	doc := document.NewDocument(document.PageSizeA4)
	if err := doc.AddHTMLTemplate(string(htmlBytes), data, nil); err != nil {
		return nil, fmt.Errorf("failed to render HTML template to folio: %v", err)
	}

	// 5. Output PDF bytes
	return doc.ToBytes()
}

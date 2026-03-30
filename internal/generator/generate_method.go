package generator

import (
	"awb-gen/internal/awb"
)

// RenderLabel performs the CPU-heavy per-label work: barcode encoding and PNG
// compression. It does NOT touch gofpdf or produce a PDF.
//
// On barcode failure the result carries a nil BarcodePNG and non-nil Err.
// The assembler will draw a text placeholder instead of an image.
func (g *MarotoGenerator) RenderLabel(record awb.AWB) ([]byte, error) {
	png, err := g.renderBarcodePNG(record.AWBNumber)
	return png, err
}
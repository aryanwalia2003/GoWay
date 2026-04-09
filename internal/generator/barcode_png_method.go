package generator

import "awb-gen/internal/barcode"

// renderBarcodePNG encodes the AWB number into a PNG byte slice.
func (g *MarotoGenerator) renderBarcodePNG(content string) ([]byte, error) {
	return barcode.RenderBarcodePNG(g.barcodeEncoder, content)
}

// renderBarcodeFallbackText returns a placeholder string when barcode encoding fails.
func renderBarcodeFallbackText(awbNumber string) string {
	return "[BARCODE: " + awbNumber + "]"
}

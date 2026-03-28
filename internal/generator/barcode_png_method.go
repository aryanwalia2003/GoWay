package generator

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

// Barcode raster constants — tuned for thermal-printer scanner legibility.
const (
	barcodeBarWidthPx = 3  // pixels wide per bar module (3px = crisp at 203 dpi)
	barcodeImgHeightPx = 72 // total image height in pixels
	barcodePaddingPx   = 6  // silent-zone padding top & bottom inside image
)

// renderBarcodePNG converts the bar-pattern produced by the barcode.Renderer
// into a PNG byte slice suitable for embedding in a maroto image component.
//
// The image is rasterised at barcodeBarWidthPx pixels per bar module and
// barcodeImgHeightPx pixels tall. This gives adequate scanner resolution for
// thermal label printers at 203–300 dpi without excessive memory use.
//
// Memory layout: one image.NRGBA allocation per call, released to GC when the
// PDF byte slice is handed off — no retained state.
func (g *MarotoGenerator) renderBarcodePNG(content string) ([]byte, error) {
	bars, barCount, err := g.barcodeEncoder.Encode(content)
	if err != nil {
		return nil, fmt.Errorf("renderBarcodePNG %q: %w", content, err)
	}

	imgW := barCount * barcodeBarWidthPx
	imgH := barcodeImgHeightPx
	barTop := barcodePaddingPx
	barBot := imgH - barcodePaddingPx

	img := image.NewNRGBA(image.Rect(0, 0, imgW, imgH))

	// Flood fill with white.
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.NRGBA{A: 255}
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			img.SetNRGBA(x, y, white)
		}
	}

	// Paint black bar columns.
	for module, isBlack := range bars {
		if !isBlack {
			continue
		}
		xLeft := module * barcodeBarWidthPx
		for dx := 0; dx < barcodeBarWidthPx; dx++ {
			for y := barTop; y < barBot; y++ {
				img.SetNRGBA(xLeft+dx, y, black)
			}
		}
	}

	// Encode to PNG. Pre-grow buffer to reduce reallocation inside png.Encode.
	var buf bytes.Buffer
	buf.Grow(imgW * imgH / 3)
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("renderBarcodePNG png.Encode: %w", err)
	}
	return buf.Bytes(), nil
}
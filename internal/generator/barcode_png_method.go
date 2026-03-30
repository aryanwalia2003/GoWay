package generator

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"sync"
)

const (
	barcodeBarWidthPx  = 3
	barcodeImgHeightPx = 72
	barcodePaddingPx   = 6
)

// pngBufPool recycles encode buffers across RenderLabel calls.
// Each worker owns its own MarotoGenerator so pool access is single-threaded
// per goroutine — no contention, just avoiding per-label heap allocation.
var pngBufPool = sync.Pool{
	New: func() any {
		b := &bytes.Buffer{}
		b.Grow(3 * 1024)
		return b
	},
}

// renderBarcodePNG encodes the AWB number into a PNG byte slice.
// The pixel fill uses direct Pix slice writes instead of SetNRGBA to avoid
// per-pixel bounds checks and function call overhead.
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

	// Flood fill white via raw Pix slice — avoids per-pixel bounds check overhead.
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255   // R
		img.Pix[i+1] = 255 // G
		img.Pix[i+2] = 255 // B
		img.Pix[i+3] = 255 // A
	}

	// Paint black bar columns.
	for module, isBlack := range bars {
		if !isBlack {
			continue
		}
		xLeft := module * barcodeBarWidthPx
		for dx := 0; dx < barcodeBarWidthPx; dx++ {
			x := xLeft + dx
			for y := barTop; y < barBot; y++ {
				idx := img.PixOffset(x, y)
				img.Pix[idx] = 0   // R
				img.Pix[idx+1] = 0 // G
				img.Pix[idx+2] = 0 // B
				img.Pix[idx+3] = 255 // A
			}
		}
	}

	buf := pngBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	if err := png.Encode(buf, img); err != nil {
		pngBufPool.Put(buf)
		return nil, fmt.Errorf("renderBarcodePNG png.Encode: %w", err)
	}

	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	pngBufPool.Put(buf)
	return out, nil
}

// renderBarcodeFallbackText returns a placeholder string when barcode encoding fails.
func renderBarcodeFallbackText(awbNumber string) string {
	return "[BARCODE: " + awbNumber + "]"
}
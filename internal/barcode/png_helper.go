package barcode

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"sync"
)

const (
	BarcodeBarWidthPx  = 3
	BarcodeImgHeightPx = 150
	BarcodePaddingPx   = 6
)

// PngBufPool recycles encode buffers.
var PngBufPool = sync.Pool{
	New: func() any {
		b := &bytes.Buffer{}
		b.Grow(3 * 1024)
		return b
	},
}

// RenderBarcodePNG takes a renderer and encodes the content into a PNG byte slice.
func RenderBarcodePNG(renderer Renderer, content string) ([]byte, error) {
	bars, barCount, err := renderer.Encode(content)
	if err != nil {
		return nil, fmt.Errorf("RenderBarcodePNG %q: %w", content, err)
	}

	imgW := barCount * BarcodeBarWidthPx
	imgH := BarcodeImgHeightPx
	barTop := BarcodePaddingPx
	barBot := imgH - BarcodePaddingPx

	img := image.NewNRGBA(image.Rect(0, 0, imgW, imgH))

	// Flood fill white via raw Pix slice
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
		xLeft := module * BarcodeBarWidthPx
		for dx := 0; dx < BarcodeBarWidthPx; dx++ {
			x := xLeft + dx
			for y := barTop; y < barBot; y++ {
				idx := img.PixOffset(x, y)
				img.Pix[idx] = 0     // R
				img.Pix[idx+1] = 0   // G
				img.Pix[idx+2] = 0   // B
				img.Pix[idx+3] = 255 // A
			}
		}
	}

	buf := PngBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(buf, img); err != nil {
		PngBufPool.Put(buf)
		return nil, fmt.Errorf("RenderBarcodePNG png.Encode: %w", err)
	}

	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	PngBufPool.Put(buf)
	return out, nil
}

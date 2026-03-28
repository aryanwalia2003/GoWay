package barcode

import (
	"fmt"
	"image/color"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
)

// Encode produces a flat bool slice representing the barcode bars left-to-right.
// true == black bar, false == white space.
// width is the number of logical columns in the barcode image.
//
// The boombuler library generates an image.Image; we walk pixel column 0 (any
// row works for 1-D barcodes) to extract the bar pattern without allocating a
// full rasterised image — we scale to 1×1 height to keep alloc minimal.
func (e *Code128Encoder) Encode(content string) (bars []bool, width int, err error) {
	raw, encErr := code128.Encode(content)
	if encErr != nil {
		return nil, 0, fmt.Errorf("barcode encode %q: %w", content, encErr)
	}

	// Scale to minimal height (1px) — we only need column data.
	scaled, scaleErr := barcode.Scale(raw, raw.Bounds().Dx(), 1)
	if scaleErr != nil {
		return nil, 0, fmt.Errorf("barcode scale %q: %w", content, scaleErr)
	}

	bounds := scaled.Bounds()
	width = bounds.Dx()
	bars = make([]bool, width)

	black := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	_ = black // used implicitly via luminance check below

	for x := 0; x < width; x++ {
		r, g, b, _ := scaled.At(x+bounds.Min.X, bounds.Min.Y).RGBA()
		// luminance: dark pixel == bar
		luma := (r*299 + g*587 + b*114) / 1000
		bars[x] = luma < 32768 // below 50% brightness == black bar
	}

	return bars, width, nil
}

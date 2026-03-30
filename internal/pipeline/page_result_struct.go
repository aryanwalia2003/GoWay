package pipeline

import "awb-gen/internal/awb"

// RenderResult is the output of a worker goroutine for a single label.
// Workers perform only the CPU-heavy work: barcode encoding and PNG compression.
// The assembler goroutine receives these and draws them into a single gofpdf document.
type RenderResult struct {
	Index      int     // original position in input — assembler uses this for ordering
	Record     awb.AWB // full AWB data for text layout
	BarcodePNG []byte  // pre-compressed PNG — workers pay the zlib cost, not the assembler
	Err        error   // non-nil if barcode encoding failed; assembler skips and logs
}
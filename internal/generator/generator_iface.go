package generator

import (
	"awb-gen/internal/awb"
)

// LabelGenerator is the contract for the CPU-intensive per-label work:
// barcode encoding and PNG compression. It does NOT produce a PDF.
// The assembler goroutine handles all PDF drawing from these results.
//
// Each goroutine must own its own LabelGenerator instance — never shared.
type LabelGenerator interface {
	RenderLabel(record awb.AWB) ([]byte, error)
}
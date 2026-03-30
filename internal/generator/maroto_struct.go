package generator

import "awb-gen/internal/barcode"

// MarotoGenerator handles the CPU-intensive per-label work: barcode encoding
// and PNG rasterisation. It no longer produces PDFs — the assembler does that.
//
// Each pipeline worker owns its own instance. Never shared between goroutines.
type MarotoGenerator struct {
	barcodeEncoder barcode.Renderer
	// Font bytes retained so the assembler ctor can read them from any worker
	// instance — they are read-only after construction.
	RegularFont []byte
	BoldFont    []byte
}
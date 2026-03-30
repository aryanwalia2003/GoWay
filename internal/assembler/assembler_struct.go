package assembler

import "go.uber.org/zap"

// GofpdfAssembler implements Assembler using a single gofpdf document.
// It receives pre-rendered barcode PNGs and AWB records from workers,
// draws each label page directly, and streams to disk via gofpdf's native
// output — eliminating pdfcpu and all xref merging entirely.
type GofpdfAssembler struct {
	log         *zap.Logger
	regularFont []byte
	boldFont    []byte
}

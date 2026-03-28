package generator

import "awb-gen/internal/barcode"

// MarotoGenerator renders AWB labels using the maroto v2 PDF library.
//
// Each instance is fully independent. It must NEVER be shared between
// goroutines. Pipeline workers each construct their own via NewMarotoGenerator.
//
// regularFont and boldFont are read-only references to the package-level
// go:embed byte slices in internal/assets — safe to share the pointer,
// never written after construction.
type MarotoGenerator struct {
	barcodeEncoder barcode.Renderer
	regularFont    []byte
	boldFont       []byte
}
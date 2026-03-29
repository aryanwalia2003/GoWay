package generator

import (
	"awb-gen/internal/assets"
	"awb-gen/internal/barcode"
)

// NewMarotoGenerator returns a MarotoGenerator ready to render labels.
// The font byte slices are copied deeply so that the underlying pdf engine
// (gofpdf) does not mutate shared embedded byte arrays during in-place
// checksum generation across multiple goroutines.
func NewMarotoGenerator(enc barcode.Renderer) *MarotoGenerator {
	regCopy := make([]byte, len(assets.RobotoRegular))
	copy(regCopy, assets.RobotoRegular)

	boldCopy := make([]byte, len(assets.RobotoBold))
	copy(boldCopy, assets.RobotoBold)

	return &MarotoGenerator{
		barcodeEncoder: enc,
		regularFont:    regCopy,
		boldFont:       boldCopy,
	}
}
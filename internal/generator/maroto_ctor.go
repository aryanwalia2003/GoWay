package generator

import (
	"awb-gen/internal/assets"
	"awb-gen/internal/barcode"
)

// NewMarotoGenerator returns a MarotoGenerator ready to render barcodes.
// Font bytes are copied so gofpdf internals cannot mutate the shared embed.
func NewMarotoGenerator(enc barcode.Renderer) *MarotoGenerator {
	regCopy := make([]byte, len(assets.RobotoRegular))
	copy(regCopy, assets.RobotoRegular)

	boldCopy := make([]byte, len(assets.RobotoBold))
	copy(boldCopy, assets.RobotoBold)

	return &MarotoGenerator{
		barcodeEncoder: enc,
		RegularFont:    regCopy,
		BoldFont:       boldCopy,
	}
}
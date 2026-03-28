package generator

import (
	"awb-gen/internal/assets"
	"awb-gen/internal/barcode"
)

// NewMarotoGenerator returns a MarotoGenerator ready to render labels.
// The font byte slices are read-only references to the embedded asset data;
// no copy is made. The returned generator must not be shared across goroutines.
func NewMarotoGenerator(enc barcode.Renderer) *MarotoGenerator {
	return &MarotoGenerator{
		barcodeEncoder: enc,
		regularFont:    assets.RobotoRegular,
		boldFont:       assets.RobotoBold,
	}
}
package generator

import (
	"awb-gen/internal/assets"
	"awb-gen/internal/barcode"

	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/orientation"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// NewMarotoGenerator returns a MarotoGenerator ready to render labels.
//
// Font bytes are copied so gofpdf cannot mutate the shared embedded slices
// during checksum generation across goroutines.
//
// The Maroto config is built once here and reused on every GenerateLabel call.
// This eliminates the config.NewBuilder chain, customFonts allocation, and
// font registration that previously ran per-label.
func NewMarotoGenerator(enc barcode.Renderer) *MarotoGenerator {
	regCopy := make([]byte, len(assets.RobotoRegular))
	copy(regCopy, assets.RobotoRegular)

	boldCopy := make([]byte, len(assets.RobotoBold))
	copy(boldCopy, assets.RobotoBold)

	fonts := []*entity.CustomFont{
		{Family: FontFamily, Style: fontstyle.Normal, Bytes: regCopy},
		{Family: FontFamily, Style: fontstyle.Bold, Bytes: boldCopy},
	}

	cfg := config.NewBuilder().
		WithPageSize(pagesize.A6).
		WithOrientation(orientation.Horizontal).
		WithLeftMargin(MarginMM).
		WithRightMargin(MarginMM).
		WithTopMargin(MarginMM).
		WithBottomMargin(MarginMM).
		WithCustomFonts(fonts).
		WithDefaultFont(&props.Font{
			Family: FontFamily,
			Style:  fontstyle.Normal,
			Size:   FontSizeNormal,
		}).
		Build()

	return &MarotoGenerator{
		barcodeEncoder: enc,
		regularFont:    regCopy,
		boldFont:       boldCopy,
		cfg:            cfg,
	}
}
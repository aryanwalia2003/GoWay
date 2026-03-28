package generator

import (
	"awb-gen/internal/awb"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/orientation"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// estimatedLabelBytes is 5 KB (pre-allocated to avoid slice growth).
const estimatedLabelBytes = 5 * 1024

// GenerateLabel renders a single AWB record into a self-contained PDF.
func (g *MarotoGenerator) GenerateLabel(record awb.AWB) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A6).
		WithOrientation(orientation.Horizontal).
		WithLeftMargin(MarginMM).
		WithRightMargin(MarginMM).
		WithTopMargin(MarginMM).
		WithBottomMargin(MarginMM).
		WithCustomFonts(g.customFonts()).
		WithDefaultFont(&props.Font{
			Family: FontFamily,
			Style:  fontstyle.Normal,
			Size:   FontSizeNormal,
		}).
		Build()

	m := maroto.New(cfg)

	addAllRows(m, record, g)

	doc, err := m.Generate()
	if err != nil {
		return nil, err
	}

	// Pre-allocate the output buffer for efficiency.
	buf := make([]byte, 0, estimatedLabelBytes)

	raw := doc.GetBytes()
	buf = append(buf, raw...)

	return buf, nil
}

// customFonts returns the font weights required by the generator.
func (g *MarotoGenerator) customFonts() []*entity.CustomFont {
	return []*entity.CustomFont{
		{
			Family: FontFamily,
			Style:  fontstyle.Normal,
			Bytes:  g.regularFont,
		},
		{
			Family: FontFamily,
			Style:  fontstyle.Bold,
			Bytes:  g.boldFont,
		},
	}
}

// addAllRows assembles the full AWB layout.
func addAllRows(m core.Maroto, r awb.AWB, g *MarotoGenerator) {
	addHeaderRows(m, r)
	addAddressRows(m, r)
	addBarcodeRows(m, r, g)
	addSKURow(m, r)
	addFooterRow(m, r)
}

// ── Header ────────────────────────────────────────────────────────────────────

func addHeaderRows(m core.Maroto, r awb.AWB) {
	m.AddRows(
		row.New(8).Add(
			col.New(ColHalf).Add(
				text.New("AWB: "+r.AWBNumber, props.Text{
					Family: FontFamily,
					Style:  fontstyle.Bold,
					Size:   FontSizeTitle,
					Align:  align.Left,
					Top:    1,
				}),
			),
			col.New(ColHalf).Add(
				text.New("Order: "+r.OrderID, props.Text{
					Family: FontFamily,
					Size:   FontSizeNormal,
					Align:  align.Right,
					Top:    1,
				}),
			),
		),
		row.New(6).Add(
			col.New(ColFull).Add(
				text.New("From: "+r.Sender, props.Text{
					Family: FontFamily,
					Size:   FontSizeNormal,
					Align:  align.Left,
					Top:    1,
				}),
			),
		),
		row.New(1).Add(col.New(ColFull).Add(line.New(props.Line{}))),
	)
}

// ── Address ───────────────────────────────────────────────────────────────────

func addAddressRows(m core.Maroto, r awb.AWB) {
	m.AddRows(
		row.New(7).Add(
			col.New(ColFull).Add(
				text.New("To: "+r.Receiver, props.Text{
					Family: FontFamily,
					Style:  fontstyle.Bold,
					Size:   FontSizeTitle,
					Align:  align.Left,
					Top:    1,
				}),
			),
		),
		row.New(10).Add(
			col.New(ColTwo3).Add(
				text.New(r.Address, props.Text{
					Family: FontFamily,
					Size:   FontSizeNormal,
					Align:  align.Left,
					Top:    1,
				}),
			),
			col.New(ColOne3).Add(
				text.New("PIN: "+r.Pincode+"\nWt:  "+r.Weight, props.Text{
					Family: FontFamily,
					Size:   FontSizeNormal,
					Align:  align.Right,
					Top:    1,
				}),
			),
		),
		row.New(1).Add(col.New(ColFull).Add(line.New(props.Line{}))),
	)
}

// ── Barcode ───────────────────────────────────────────────────────────────────

func addBarcodeRows(m core.Maroto, r awb.AWB, g *MarotoGenerator) {
	pngBytes, err := g.renderBarcodePNG(r.AWBNumber)
	if err != nil {
		// Graceful degradation: failed barcode → styled text placeholder.
		m.AddRows(
			row.New(BarcodeHeightMM).Add(
				col.New(ColFull).Add(
					text.New("[BARCODE: "+r.AWBNumber+"]", props.Text{
						Family: FontFamily,
						Style:  fontstyle.Bold,
						Size:   FontSizeNormal,
						Align:  align.Center,
						Top:    5,
					}),
				),
			),
		)
		return
	}

	m.AddRows(
		row.New(BarcodeHeightMM).Add(
			col.New(ColFull).Add(
				image.NewFromBytes(pngBytes, extension.Png, props.Rect{
					Center:  true,
					Percent: 92,
				}),
			),
		),
		row.New(5).Add(
			col.New(ColFull).Add(
				text.New(r.AWBNumber, props.Text{
					Family: FontFamily,
					Size:   FontSizeSmall,
					Align:  align.Center,
				}),
			),
		),
	)
}

// ── SKU ───────────────────────────────────────────────────────────────────────

func addSKURow(m core.Maroto, r awb.AWB) {
	m.AddRows(
		row.New(1).Add(col.New(ColFull).Add(line.New(props.Line{}))),
		row.New(6).Add(
			col.New(ColFull).Add(
				text.New("Items: "+r.SKUDetails, props.Text{
					Family: FontFamily,
					Size:   FontSizeSmall,
					Align:  align.Left,
					Top:    1,
				}),
			),
		),
	)
}

// ── Footer ────────────────────────────────────────────────────────────────────

func addFooterRow(m core.Maroto, r awb.AWB) {
	m.AddRows(
		row.New(5).Add(
			col.New(ColHalf).Add(
				text.New("Weight: "+r.Weight, props.Text{
					Family: FontFamily,
					Size:   FontSizeSmall,
					Align:  align.Left,
					Top:    1,
				}),
			),
			col.New(ColHalf).Add(
				text.New("awb-gen", props.Text{
					Family: FontFamily,
					Size:   FontSizeSmall,
					Align:  align.Right,
					Top:    1,
				}),
			),
		),
	)
}
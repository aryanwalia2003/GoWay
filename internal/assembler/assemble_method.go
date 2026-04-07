package assembler

import (
	"bytes"
	"container/heap"
	"fmt"
	"io"
	"strings"

	"awb-gen/internal/pipeline"

	"github.com/phpdave11/gofpdf"
	"go.uber.org/zap"
)

const (
	fontFamily = "roboto"
	marginMM   = 4.0
	// A6 landscape: 148mm wide × 105mm tall
	pageW = 148.0
	pageH = 105.0

	fontSizeTitle  = 9.0
	fontSizeNormal = 7.0
	fontSizeSmall  = 6.0

	barcodeHeightMM = 16.0
)

// AssembleToFile consumes RenderResults in arrival order, buffers them via a
// min-heap to restore input ordering, then draws each label page into a single
// gofpdf document streamed directly to outPath.
//
// Fonts are registered once on the gofpdf instance at startup.
// Each page costs: one AddPage() + a handful of Cell/Image calls.
// No pdfcpu. No xref merging. No temp files. Structurally valid by construction.
func (a *GofpdfAssembler) AssembleToFile(results <-chan pipeline.RenderResult, outPath string) (int, error) {
	return a.assemble(results, func(pdf *gofpdf.Fpdf) error {
		if err := pdf.OutputFileAndClose(outPath); err != nil {
			return fmt.Errorf("assembler: write output %q: %w", outPath, err)
		}
		return nil
	})
}

// AssembleToWriter behaves the same as AssembleToFile but writes the PDF output directly to an io.Writer.
func (a *GofpdfAssembler) AssembleToWriter(results <-chan pipeline.RenderResult, w io.Writer) (int, error) {
	return a.assemble(results, func(pdf *gofpdf.Fpdf) error {
		if err := pdf.Output(w); err != nil {
			return fmt.Errorf("assembler: write stream output: %w", err)
		}
		return nil
	})
}

func (a *GofpdfAssembler) assemble(results <-chan pipeline.RenderResult, finish func(*gofpdf.Fpdf) error) (int, error) {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr:    "mm",
		Size:       gofpdf.SizeType{Wd: pageW, Ht: pageH},
		FontDirStr: "",
	})
	pdf.SetMargins(marginMM, marginMM, marginMM)
	pdf.SetAutoPageBreak(false, 0)

	pdf.AddUTF8FontFromBytes(fontFamily, "", a.regularFont)
	pdf.AddUTF8FontFromBytes(fontFamily, "B", a.boldFont)

	if pdf.Err() {
		return 0, fmt.Errorf("assembler: gofpdf init error: %s", pdf.Error())
	}

	h := &renderHeap{}
	heap.Init(h)
	nextIndex := 0
	drawn := 0

	drawPage := func(r pipeline.RenderResult) error {
		pdf.AddPage()
		if pdf.Err() {
			return fmt.Errorf("assembler: AddPage (index=%d): %s", r.Index, pdf.Error())
		}
		drawLabel(pdf, r)
		drawn++
		return nil
	}

	drainHeap := func() error {
		for h.Len() > 0 && (*h)[0].Index == nextIndex {
			r := heap.Pop(h).(pipeline.RenderResult)
			nextIndex++
			if r.Err != nil && r.BarcodePNG == nil {
			}
			if err := drawPage(r); err != nil {
				return err
			}
		}
		return nil
	}

	for r := range results {
		if r.Err != nil {
			a.log.Warn("assembler: barcode encode failed, using text fallback",
				zap.Int("index", r.Index),
				zap.String("awb_number", r.Record.AWBNumber),
				zap.Error(r.Err),
			)
		}
		heap.Push(h, r)
		if err := drainHeap(); err != nil {
			return 0, err
		}
	}

	for h.Len() > 0 {
		r := heap.Pop(h).(pipeline.RenderResult)
		nextIndex++
		if err := drawPage(r); err != nil {
			return 0, err
		}
	}

	if drawn == 0 {
		return 0, fmt.Errorf("assembler: no pages were drawn")
	}

	if err := finish(pdf); err != nil {
		return 0, err
	}

	a.log.Info("assembler: complete", zap.Int("pages", drawn))
	return drawn, nil
}

// drawLabel draws one AWB label page onto the current gofpdf page.
// The page has already been added by the caller.
// Layout mirrors the original Maroto layout exactly.
func drawLabel(pdf *gofpdf.Fpdf, r pipeline.RenderResult) {
	rec := r.Record
	contentW := pageW - 2*marginMM // usable width inside margins

	pdf.SetFont(fontFamily, "B", fontSizeTitle)
	pdf.SetXY(marginMM, marginMM)

	// ── Row 1: AWB number (left) + Order ID (right) ───────────────────────────
	half := contentW / 2
	pdf.CellFormat(half, 8, "AWB: "+rec.AWBNumber, "", 0, "L", false, 0, "")
	pdf.SetFont(fontFamily, "", fontSizeNormal)
	pdf.CellFormat(half, 8, "Order: "+rec.OrderID, "", 1, "R", false, 0, "")

	// ── Row 2: Sender ─────────────────────────────────────────────────────────
	pdf.SetXY(marginMM, pdf.GetY())
	pdf.CellFormat(contentW, 6, "From: "+rec.Sender, "", 1, "L", false, 0, "")

	// ── Divider ───────────────────────────────────────────────────────────────
	y := pdf.GetY()
	pdf.Line(marginMM, y, pageW-marginMM, y)
	pdf.SetY(y + 1)

	// ── Row 3: Receiver ───────────────────────────────────────────────────────
	pdf.SetFont(fontFamily, "B", fontSizeTitle)
	pdf.SetXY(marginMM, pdf.GetY())
	pdf.CellFormat(contentW, 7, "To: "+rec.Receiver, "", 1, "L", false, 0, "")

	// ── Row 4: Address (left 2/3) + PIN/Weight (right 1/3) ───────────────────
	pdf.SetFont(fontFamily, "", fontSizeNormal)
	pdf.SetXY(marginMM, pdf.GetY())
	addrW := contentW * 2 / 3
	pinW := contentW - addrW

	// MultiCell for address to handle wrapping within its column.
	addrX := pdf.GetX()
	addrY := pdf.GetY()
	pdf.MultiCell(addrW, 4, rec.Address, "", "L", false)
	addrEndY := pdf.GetY()

	// PIN + Weight in right column, aligned to addrY.
	pdf.SetXY(addrX+addrW, addrY)
	pinText := "PIN: " + rec.Pincode + "\nWt:  " + rec.Weight
	for _, line := range strings.Split(pinText, "\n") {
		pdf.CellFormat(pinW, 4, line, "", 1, "R", false, 0, "")
		pdf.SetX(addrX + addrW)
	}

	// Advance past whichever column was taller.
	if pdf.GetY() < addrEndY {
		pdf.SetY(addrEndY)
	}

	// ── Divider ───────────────────────────────────────────────────────────────
	y = pdf.GetY()
	pdf.Line(marginMM, y, pageW-marginMM, y)
	pdf.SetY(y + 1)

	// ── Row 5: Barcode image or fallback text ─────────────────────────────────
	barcodeY := pdf.GetY()
	if r.BarcodePNG != nil {
		imgOpts := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}
		pdf.RegisterImageOptionsReader(
			rec.AWBNumber, // unique key per label
			imgOpts,
			bytes.NewReader(r.BarcodePNG),
		)
		// Centre the image horizontally within the content area.
		imgW := contentW * 0.92
		imgX := marginMM + (contentW-imgW)/2
		pdf.ImageOptions(rec.AWBNumber, imgX, barcodeY, imgW, barcodeHeightMM, false, imgOpts, 0, "")
	} else {
		pdf.SetFont(fontFamily, "B", fontSizeNormal)
		pdf.SetXY(marginMM, barcodeY)
		pdf.CellFormat(contentW, barcodeHeightMM, "[BARCODE: "+rec.AWBNumber+"]", "", 0, "C", false, 0, "")
	}
	pdf.SetY(barcodeY + barcodeHeightMM)

	// ── Row 6: AWB number text under barcode ──────────────────────────────────
	pdf.SetFont(fontFamily, "", fontSizeSmall)
	pdf.SetXY(marginMM, pdf.GetY())
	pdf.CellFormat(contentW, 5, rec.AWBNumber, "", 1, "C", false, 0, "")

	// ── Divider ───────────────────────────────────────────────────────────────
	y = pdf.GetY()
	pdf.Line(marginMM, y, pageW-marginMM, y)
	pdf.SetY(y + 1)

	// ── Row 7: SKU details ────────────────────────────────────────────────────
	pdf.SetXY(marginMM, pdf.GetY())
	pdf.CellFormat(contentW, 6, "Items: "+rec.SKUDetails, "", 1, "L", false, 0, "")

	// ── Row 8: Footer — weight (left) + branding (right) ─────────────────────
	pdf.SetXY(marginMM, pdf.GetY())
	pdf.CellFormat(half, 5, "Weight: "+rec.Weight, "", 0, "L", false, 0, "")
	pdf.CellFormat(half, 5, "awb-gen", "", 1, "R", false, 0, "")
}

// ── renderHeap ───────────────────────────────────────────────────────────────

type renderHeap []pipeline.RenderResult

func (h renderHeap) Len() int           { return len(h) }
func (h renderHeap) Less(i, j int) bool { return h[i].Index < h[j].Index }
func (h renderHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *renderHeap) Push(x any) { *h = append(*h, x.(pipeline.RenderResult)) }
func (h *renderHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = pipeline.RenderResult{}
	*h = old[:n-1]
	return x
}

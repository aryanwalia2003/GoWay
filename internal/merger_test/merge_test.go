package merger_test

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"awb-gen/internal/awb"
	"awb-gen/internal/barcode"
	"awb-gen/internal/generator"
	"awb-gen/internal/merger"
	"awb-gen/internal/pipeline"

	"go.uber.org/zap"
)

var (
	validPDFBytes []byte
	initPDFOnce   sync.Once
)

func getValidPDF(t *testing.T) []byte {
	initPDFOnce.Do(func() {
		gen := generator.NewMarotoGenerator(barcode.NewCode128Encoder())
		pdf, err := gen.GenerateLabel(awb.AWB{
			AWBNumber: "TEST1234",
			OrderID:   "#1",
			Sender:    "S",
			Receiver:  "R",
			Address:   "A",
			Pincode:   "P",
			Weight:    "1kg",
		})
		if err != nil {
			panic("failed to generate valid PDF: " + err.Error())
		}
		validPDFBytes = pdf
	})
	return validPDFBytes
}

func makeResultChan(results []pipeline.PageResult) <-chan pipeline.PageResult {
	ch := make(chan pipeline.PageResult, len(results))
	for _, r := range results {
		ch <- r
	}
	close(ch)
	return ch
}

// newMerger returns an OrderedMerger with a small chunk size for tests so
// chunked-merge code paths are exercised even on tiny fixtures.
func newMerger() *merger.OrderedMerger {
	return merger.NewOrderedMerger(zap.NewNop(), 2)
}

func outPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "out.pdf")
}

func assertValidPDF(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("output PDF not found at %q: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("output PDF at %q is empty", path)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open output PDF: %v", err)
	}
	defer f.Close()
	hdr := make([]byte, 4)
	if _, err := f.Read(hdr); err != nil {
		t.Fatalf("read PDF header: %v", err)
	}
	if string(hdr) != "%PDF" {
		t.Fatalf("output does not start with %%PDF magic, got: %q", hdr)
	}
}

func TestOrderedMerger_MergesInOrder(t *testing.T) {
	t.Parallel()

	// Send results out of order — merger must restore index order.
	validPDF := getValidPDF(t)
	results := []pipeline.PageResult{
		{Index: 2, PDFBytes: validPDF},
		{Index: 0, PDFBytes: validPDF},
		{Index: 1, PDFBytes: validPDF},
	}

	mg := newMerger()
	path := outPath(t)
	count, err := mg.MergeToFile(makeResultChan(results), path)
	if err != nil {
		t.Fatalf("MergeToFile returned unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 pages merged, got %d", count)
	}
	assertValidPDF(t, path)
}

func TestOrderedMerger_SkipsFailedPages(t *testing.T) {
	t.Parallel()

	renderErr := errors.New("render failed")
	validPDF := getValidPDF(t)
	results := []pipeline.PageResult{
		{Index: 0, PDFBytes: validPDF},
		{Index: 1, Err: renderErr},            // should be skipped
		{Index: 2, PDFBytes: validPDF},
	}

	mg := newMerger()
	path := outPath(t)
	count, err := mg.MergeToFile(makeResultChan(results), path)
	if err != nil {
		t.Fatalf("MergeToFile returned unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 pages (1 skipped), got %d", count)
	}
	assertValidPDF(t, path)
}

func TestOrderedMerger_EmptyChannel_ReturnsError(t *testing.T) {
	t.Parallel()

	mg := newMerger()
	ch := make(chan pipeline.PageResult)
	close(ch)

	_, err := mg.MergeToFile(ch, outPath(t))
	if err == nil {
		t.Fatal("expected error for empty result channel, got nil")
	}
}

func TestOrderedMerger_ChunkBoundary(t *testing.T) {
	t.Parallel()

	// With chunkSize=2, 5 pages exercises multiple chunk-flush cycles.
	validPDF := getValidPDF(t)
	results := make([]pipeline.PageResult, 5)
	for i := range results {
		results[i] = pipeline.PageResult{Index: i, PDFBytes: validPDF}
	}

	mg := merger.NewOrderedMerger(zap.NewNop(), 2)
	path := outPath(t)
	count, err := mg.MergeToFile(makeResultChan(results), path)
	if err != nil {
		t.Fatalf("MergeToFile (chunk boundary) error: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 pages, got %d", count)
	}
	assertValidPDF(t, path)
}

func TestOrderedMerger_ChunkSizeEqualsBatch(t *testing.T) {
	t.Parallel()

	// ChunkSize == len(results): single chunk, same path as old concat() behaviour.
	validPDF := getValidPDF(t)
	results := make([]pipeline.PageResult, 3)
	for i := range results {
		results[i] = pipeline.PageResult{Index: i, PDFBytes: validPDF}
	}

	mg := merger.NewOrderedMerger(zap.NewNop(), 10) // chunk > batch
	path := outPath(t)
	count, err := mg.MergeToFile(makeResultChan(results), path)
	if err != nil {
		t.Fatalf("MergeToFile (single chunk) error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 pages, got %d", count)
	}
	assertValidPDF(t, path)
}
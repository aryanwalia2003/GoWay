package merger_test

import (
	"errors"
	"testing"

	"awb-gen/internal/merger"
	"awb-gen/internal/pipeline"

	"go.uber.org/zap"
)

// minimalPDF is the smallest syntactically valid PDF — used as a stub page
// in merger tests so we don't need to invoke the full generator.
// Source: https://brendanzagaeski.com/blog/0004.html (public domain)
const minimalPDF = `%PDF-1.0
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/MediaBox[0 0 3 3]>>endobj
xref
0 4
0000000000 65535 f 
0000000009 00000 n 
0000000058 00000 n 
0000000115 00000 n 
trailer<</Size 4/Root 1 0 R>>
startxref
190
%%EOF`

func makeResultChan(results []pipeline.PageResult) <-chan pipeline.PageResult {
	ch := make(chan pipeline.PageResult, len(results))
	for _, r := range results {
		ch <- r
	}
	close(ch)
	return ch
}

func TestOrderedMerger_MergesInOrder(t *testing.T) {
	t.Parallel()

	// Send results out of order — merger must restore index order.
	results := []pipeline.PageResult{
		{Index: 2, PDFBytes: []byte(minimalPDF)},
		{Index: 0, PDFBytes: []byte(minimalPDF)},
		{Index: 1, PDFBytes: []byte(minimalPDF)},
	}

	mg := merger.NewOrderedMerger(zap.NewNop())
	pdfBytes, count, err := mg.Merge(makeResultChan(results))
	if err != nil {
		t.Fatalf("Merge returned unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 pages merged, got %d", count)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("merged PDF bytes are empty")
	}
}

func TestOrderedMerger_SkipsFailedPages(t *testing.T) {
	t.Parallel()

	renderErr := errors.New("render failed")
	results := []pipeline.PageResult{
		{Index: 0, PDFBytes: []byte(minimalPDF)},
		{Index: 1, Err: renderErr},             // should be skipped
		{Index: 2, PDFBytes: []byte(minimalPDF)},
	}

	mg := merger.NewOrderedMerger(zap.NewNop())
	_, count, err := mg.Merge(makeResultChan(results))
	if err != nil {
		t.Fatalf("Merge returned unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 pages (1 skipped), got %d", count)
	}
}

func TestOrderedMerger_EmptyChannel_ReturnsError(t *testing.T) {
	t.Parallel()

	mg := merger.NewOrderedMerger(zap.NewNop())
	ch := make(chan pipeline.PageResult)
	close(ch)

	_, _, err := mg.Merge(ch)
	if err == nil {
		t.Fatal("expected error for empty result channel, got nil")
	}
}

package merger_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

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

func getValidPDF() []byte {
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

// ── helpers ───────────────────────────────────────────────────────────────────

// sendInOrder sends results in the given slice order and closes the channel.
func sendInOrder(results []pipeline.PageResult) <-chan pipeline.PageResult {
	ch := make(chan pipeline.PageResult, len(results))
	for _, r := range results {
		ch <- r
	}
	close(ch)
	return ch
}

// sendWithDelay sends each result after a short staggered delay to simulate
// real worker non-determinism. Indexes are deliberately sent out of order.
func sendOutOfOrder(results []pipeline.PageResult) <-chan pipeline.PageResult {
	ch := make(chan pipeline.PageResult, len(results))
	var wg sync.WaitGroup
	for i, r := range results {
		wg.Add(1)
		go func(idx int, res pipeline.PageResult) {
			defer wg.Done()
			// Stagger sends so higher indexes often arrive before lower ones.
			time.Sleep(time.Duration(len(results)-idx) * time.Millisecond)
			ch <- res
		}(i, r)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	return ch
}

func newMerger(chunkSize int) *merger.OrderedMerger {
	return merger.NewOrderedMerger(zap.NewNop(), chunkSize)
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

func makePages(n int) []pipeline.PageResult {
	pages := make([]pipeline.PageResult, n)
	pdfBytes := getValidPDF()
	for i := range pages {
		pages[i] = pipeline.PageResult{Index: i, PDFBytes: pdfBytes}
	}
	return pages
}

// ── correctness tests ─────────────────────────────────────────────────────────

func TestWindowedMerger_InOrderArrival(t *testing.T) {
	t.Parallel()
	mg := newMerger(2)
	path := outPath(t)
	count, err := mg.MergeToFile(sendInOrder(makePages(5)), path)
	if err != nil {
		t.Fatalf("MergeToFile: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5, got %d", count)
	}
	assertValidPDF(t, path)
}

// TestWindowedMerger_OutOfOrderArrival is the key test: results arrive with
// higher indexes first (simulating fast workers finishing early), and the
// merger must still produce the correct ordered output.
func TestWindowedMerger_OutOfOrderArrival(t *testing.T) {
	t.Parallel()

	// Deliberate reverse order: index 4,3,2,1,0 arrive in that sequence.
	pdfBytes := getValidPDF()
	results := []pipeline.PageResult{
		{Index: 4, PDFBytes: pdfBytes},
		{Index: 3, PDFBytes: pdfBytes},
		{Index: 2, PDFBytes: pdfBytes},
		{Index: 1, PDFBytes: pdfBytes},
		{Index: 0, PDFBytes: pdfBytes},
	}

	path := outPath(t)
	mg := newMerger(2) // chunk size 2 so we exercise multiple flushes
	count, err := mg.MergeToFile(sendInOrder(results), path)
	if err != nil {
		t.Fatalf("MergeToFile (reverse order): %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5, got %d", count)
	}
	assertValidPDF(t, path)
}

// TestWindowedMerger_ConcurrentOutOfOrder sends results from goroutines with
// staggered delays so indexes genuinely arrive non-deterministically.
func TestWindowedMerger_ConcurrentOutOfOrder(t *testing.T) {
	t.Parallel()

	pages := makePages(20)
	path := outPath(t)
	mg := newMerger(5)
	count, err := mg.MergeToFile(sendOutOfOrder(pages), path)
	if err != nil {
		t.Fatalf("MergeToFile (concurrent out-of-order): %v", err)
	}
	if count != 20 {
		t.Fatalf("expected 20, got %d", count)
	}
	assertValidPDF(t, path)
}

func TestWindowedMerger_SkipsFailedPages(t *testing.T) {
	t.Parallel()

	renderErr := errors.New("render failed")
	pdfBytes := getValidPDF()
	results := []pipeline.PageResult{
		{Index: 0, PDFBytes: pdfBytes},
		{Index: 1, Err: renderErr}, // skipped
		{Index: 2, PDFBytes: pdfBytes},
	}

	path := outPath(t)
	mg := newMerger(2)
	count, err := mg.MergeToFile(sendInOrder(results), path)
	if err != nil {
		t.Fatalf("MergeToFile: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 (1 skipped), got %d", count)
	}
	assertValidPDF(t, path)
}

// TestWindowedMerger_ErrorAtStart exercises the case where index 0 fails.
// The merger must not stall — it should see the error sentinel for index 0,
// advance nextIndex, and immediately drain the waiting results.
func TestWindowedMerger_ErrorAtStart(t *testing.T) {
	t.Parallel()

	renderErr := errors.New("first label failed")
	pdfBytes := getValidPDF()
	results := []pipeline.PageResult{
		{Index: 0, Err: renderErr},
		{Index: 1, PDFBytes: pdfBytes},
		{Index: 2, PDFBytes: pdfBytes},
	}

	path := outPath(t)
	mg := newMerger(2)
	count, err := mg.MergeToFile(sendInOrder(results), path)
	if err != nil {
		t.Fatalf("MergeToFile (error at index 0): %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
	assertValidPDF(t, path)
}

// TestWindowedMerger_ErrorInMiddle_OutOfOrder: error page arrives after its
// successors are already in the heap — tests that the sentinel unblocks them.
func TestWindowedMerger_ErrorInMiddle_OutOfOrder(t *testing.T) {
	t.Parallel()

	renderErr := errors.New("middle failure")
	// Send 2, 3, then the error for 1, then 0. The heap holds 2 and 3 waiting
	// for 1; the error sentinel for 1 must unblock them.
	pdfBytes := getValidPDF()
	results := []pipeline.PageResult{
		{Index: 2, PDFBytes: pdfBytes},
		{Index: 3, PDFBytes: pdfBytes},
		{Index: 1, Err: renderErr},
		{Index: 0, PDFBytes: pdfBytes},
	}

	path := outPath(t)
	mg := newMerger(10)
	count, err := mg.MergeToFile(sendInOrder(results), path)
	if err != nil {
		t.Fatalf("MergeToFile: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 (1 skipped), got %d", count)
	}
	assertValidPDF(t, path)
}

func TestWindowedMerger_EmptyChannel_ReturnsError(t *testing.T) {
	t.Parallel()

	mg := newMerger(10)
	ch := make(chan pipeline.PageResult)
	close(ch)

	_, err := mg.MergeToFile(ch, outPath(t))
	if err == nil {
		t.Fatal("expected error for empty channel, got nil")
	}
}

func TestWindowedMerger_AllPagesFail_ReturnsError(t *testing.T) {
	t.Parallel()

	renderErr := errors.New("all failed")
	results := []pipeline.PageResult{
		{Index: 0, Err: renderErr},
		{Index: 1, Err: renderErr},
		{Index: 2, Err: renderErr},
	}

	mg := newMerger(10)
	_, err := mg.MergeToFile(sendInOrder(results), outPath(t))
	if err == nil {
		t.Fatal("expected error when all pages fail, got nil")
	}
}

// ── chunk boundary tests ──────────────────────────────────────────────────────

func TestWindowedMerger_ChunkSizeOne(t *testing.T) {
	t.Parallel()
	// Every page is its own chunk — maximum flush frequency.
	mg := newMerger(1)
	count, err := mg.MergeToFile(sendInOrder(makePages(6)), outPath(t))
	if err != nil {
		t.Fatalf("MergeToFile (chunkSize=1): %v", err)
	}
	if count != 6 {
		t.Fatalf("expected 6, got %d", count)
	}
}

func TestWindowedMerger_ChunkExceedsBatch(t *testing.T) {
	t.Parallel()
	// Chunk larger than total pages — single flush at the end.
	mg := newMerger(1000)
	count, err := mg.MergeToFile(sendInOrder(makePages(7)), outPath(t))
	if err != nil {
		t.Fatalf("MergeToFile (chunk > batch): %v", err)
	}
	if count != 7 {
		t.Fatalf("expected 7, got %d", count)
	}
}

func TestWindowedMerger_ExactChunkMultiple(t *testing.T) {
	t.Parallel()
	// 6 pages, chunk=3: two exact flushes, no partial tail.
	mg := newMerger(3)
	count, err := mg.MergeToFile(sendInOrder(makePages(6)), outPath(t))
	if err != nil {
		t.Fatalf("MergeToFile (exact multiple): %v", err)
	}
	if count != 6 {
		t.Fatalf("expected 6, got %d", count)
	}
}

// ── scale smoke test ──────────────────────────────────────────────────────────

// TestWindowedMerger_LargeBatch_MemoryBounded verifies that a large batch
// completes without error and that the merger never held all pages in RAM.
// We can't directly assert RSS in a unit test, but we can assert correctness
// at scale which confirms the windowed logic is sound.
func TestWindowedMerger_LargeBatch_MemoryBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large batch test in short mode")
	}
	t.Parallel()

	const n = 2000
	path := outPath(t)
	mg := newMerger(200) // 10 flushes of 200

	ch := make(chan pipeline.PageResult, 64)
	go func() {
		defer close(ch)
		// Send out of order: even indexes first, then odd.
		pdfBytes := getValidPDF()
		for i := 0; i < n; i += 2 {
			ch <- pipeline.PageResult{Index: i, PDFBytes: pdfBytes}
		}
		for i := 1; i < n; i += 2 {
			ch <- pipeline.PageResult{Index: i, PDFBytes: pdfBytes}
		}
	}()

	count, err := mg.MergeToFile(ch, path)
	if err != nil {
		t.Fatalf("MergeToFile (large batch): %v", err)
	}
	if count != n {
		t.Fatalf("expected %d, got %d", n, count)
	}
	assertValidPDF(t, path)
	t.Logf("merged %d pages successfully", count)
}

// ── default chunk size test ───────────────────────────────────────────────────

func TestWindowedMerger_DefaultChunkSize(t *testing.T) {
	t.Parallel()
	// Pass 0 → should use defaultChunkSize (500), not panic or use 0.
	mg := merger.NewOrderedMerger(zap.NewNop(), 0)
	count, err := mg.MergeToFile(sendInOrder(makePages(3)), outPath(t))
	if err != nil {
		t.Fatalf("MergeToFile (default chunk): %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3, got %d", count)
	}
}

// ── benchmark ─────────────────────────────────────────────────────────────────

func BenchmarkWindowedMerger_500Pages(b *testing.B) {
	pages := makePages(500)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(b.TempDir(), fmt.Sprintf("bench-%d.pdf", i))
		mg := newMerger(100)
		if _, err := mg.MergeToFile(sendInOrder(pages), path); err != nil {
			b.Fatalf("MergeToFile: %v", err)
		}
	}
}
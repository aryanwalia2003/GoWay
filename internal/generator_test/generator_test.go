package generator_test

import (
	"bytes"
	"sync"
	"testing"

	"awb-gen/internal/awb"
	"awb-gen/internal/barcode"
	"awb-gen/internal/generator"
)

func sampleAWB() awb.AWB {
	return awb.AWB{
		AWBNumber:  "ZFW123456789",
		OrderID:    "#9982",
		Sender:     "Test Store",
		Receiver:   "John Doe",
		Address:    "123 Main St, Mumbai, India",
		Pincode:    "400001",
		Weight:     "0.5kg",
		SKUDetails: "Item A x1, Item B x2",
	}
}

func newGen() generator.LabelGenerator {
	return generator.NewMarotoGenerator(barcode.NewCode128Encoder())
}

// TestGenerateLabel_ProducesValidPDF verifies the output is a non-empty PDF.
func TestGenerateLabel_ProducesValidPDF(t *testing.T) {
	t.Parallel()

	pdf, err := newGen().GenerateLabel(sampleAWB())
	if err != nil {
		t.Fatalf("GenerateLabel error: %v", err)
	}
	if len(pdf) == 0 {
		t.Fatal("GenerateLabel returned empty bytes")
	}
	// All valid PDFs begin with %PDF.
	const magic = "%PDF"
	n := len(magic)
	if len(pdf) < n || !bytes.Equal(pdf[:n], []byte(magic)) {
		t.Fatalf("output missing %%PDF header; got: %q", pdf[:min(n, len(pdf))])
	}
}

// TestGenerateLabel_DistinctOutputPerRecord ensures two different AWBs produce
// different PDFs — catches any state-leak between calls.
func TestGenerateLabel_DistinctOutputPerRecord(t *testing.T) {
	t.Parallel()

	gen := newGen()
	a := sampleAWB()
	b := sampleAWB()
	b.AWBNumber = "ZFW999999999"
	b.Receiver = "Jane Smith"

	pdfA, errA := gen.GenerateLabel(a)
	pdfB, errB := gen.GenerateLabel(b)
	if errA != nil || errB != nil {
		t.Fatalf("errors: %v / %v", errA, errB)
	}
	if bytes.Equal(pdfA, pdfB) {
		t.Fatal("two distinct AWB records produced identical PDF bytes — state leak suspected")
	}
}

// TestGenerateLabel_ConcurrentWorkers_NoRace runs 8 independent workers each
// generating 10 labels. Run with -race to surface any data races.
func TestGenerateLabel_ConcurrentWorkers_NoRace(t *testing.T) {
	t.Parallel()

	const workers = 8
	const perWorker = 10

	enc := barcode.NewCode128Encoder() // stateless — safe to share the pointer
	errCh := make(chan error, workers*perWorker)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Each goroutine owns its own MarotoGenerator — mirrors pipeline design.
			gen := generator.NewMarotoGenerator(enc)
			for i := 0; i < perWorker; i++ {
				if _, err := gen.GenerateLabel(sampleAWB()); err != nil {
					errCh <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("worker error: %v", err)
	}
}

// BenchmarkGenerateLabel measures throughput of a single goroutine rendering
// one label repeatedly. Run: go test -bench=BenchmarkGenerateLabel -benchmem
func BenchmarkGenerateLabel(b *testing.B) {
	gen := newGen()
	rec := sampleAWB()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = gen.GenerateLabel(rec)
	}
}

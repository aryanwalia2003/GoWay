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

// TestRenderLabel_ProducesValidPNG verifies the output is a non-empty PNG.
func TestRenderLabel_ProducesValidPNG(t *testing.T) {
	t.Parallel()

	pngBytes, err := newGen().RenderLabel(sampleAWB())
	if err != nil {
		t.Fatalf("RenderLabel error: %v", err)
	}
	if len(pngBytes) == 0 {
		t.Fatal("RenderLabel returned empty bytes")
	}
	// All valid PNGs begin with 8-byte signature: 89 50 4E 47 0D 0A 1A 0A
	magic := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
	n := len(magic)
	if len(pngBytes) < n || !bytes.Equal(pngBytes[:n], magic) {
		t.Fatalf("output missing PNG header; got: %x", pngBytes[:min(n, len(pngBytes))])
	}
}

// TestRenderLabel_DistinctOutputPerRecord ensures two different AWBs produce
// different PNGs — catches any state-leak between calls.
func TestRenderLabel_DistinctOutputPerRecord(t *testing.T) {
	t.Parallel()

	gen := newGen()
	a := sampleAWB()
	b := sampleAWB()
	b.AWBNumber = "ZFW999999999"
	b.Receiver = "Jane Smith"

	pngA, errA := gen.RenderLabel(a)
	pngB, errB := gen.RenderLabel(b)
	if errA != nil || errB != nil {
		t.Fatalf("errors: %v / %v", errA, errB)
	}
	if bytes.Equal(pngA, pngB) {
		t.Fatal("two distinct AWB records produced identical PNG bytes — state leak suspected")
	}
}

// TestRenderLabel_ConcurrentWorkers_NoRace runs 8 independent workers each
// generating 10 labels. Run with -race to surface any data races.
func TestRenderLabel_ConcurrentWorkers_NoRace(t *testing.T) {
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
				if _, err := gen.RenderLabel(sampleAWB()); err != nil {
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

// BenchmarkRenderLabel measures throughput of a single goroutine rendering
// one label repeatedly. Run: go test -bench=BenchmarkRenderLabel -benchmem
func BenchmarkRenderLabel(b *testing.B) {
	gen := newGen()
	rec := sampleAWB()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = gen.RenderLabel(rec)
	}
}

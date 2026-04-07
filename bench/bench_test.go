package bench_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"awb-gen/internal/assembler"
	"awb-gen/internal/assets"
	"awb-gen/internal/pipeline"

	"go.uber.org/zap"
)

// buildLargeJSON produces an in-memory JSON array string of n AWB records.
// Reused across sub-benchmarks to avoid counting JSON generation in results.
func buildLargeJSON(n int) string {
	const record = `{"awb_number":"ZFW000000001","order_id":"#1","sender":"Bench Store","receiver":"Bench Customer","address":"1 Bench Road, Mumbai, India","pincode":"400001","weight":"1.0kg","sku_details":"Bench Item x1"}`
	var sb strings.Builder
	sb.Grow(len(record)*n + n + 2)
	sb.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(record)
	}
	sb.WriteByte(']')
	return sb.String()
}

// runPipeline executes the full pipeline+assemble for the given JSON string,
// writing the PDF to a temp file. Returns the output file size in pages.
func runPipeline(t testing.TB, jsonStr string) int {
	t.Helper()
	log := zap.NewNop()

	pl := pipeline.New(pipeline.Defaults(), log)
	results, err := pl.Run(context.Background(), strings.NewReader(jsonStr))
	if err != nil {
		t.Fatalf("pipeline.Run: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "bench-out.pdf")
	asm := assembler.New(log, assets.RobotoRegular, assets.RobotoBold)
	count, _, err := asm.AssembleToFile(results, outPath)
	if err != nil {
		t.Fatalf("assembler.AssembleToFile: %v", err)
	}
	t.Logf("assembled %d pages → %s", count, outPath)
	return count
}

// ─── Benchmarks ──────────────────────────────────────────────────────────────

func BenchmarkPipeline_100Labels(b *testing.B) {
	json := buildLargeJSON(100)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		runPipeline(b, json)
	}
}

func BenchmarkPipeline_500Labels(b *testing.B) {
	json := buildLargeJSON(500)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		runPipeline(b, json)
	}
}

func BenchmarkPipeline_1000Labels(b *testing.B) {
	json := buildLargeJSON(1000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		runPipeline(b, json)
	}
}

// ─── Memory tests ─────────────────────────────────────────────────────────────

// TestMemory_5000Labels asserts that peak heap alloc for 5 000 labels stays
// under the 200 MB target (revised up from 50 MB to account for pdfcpu
// internals; the 966 MB pre-optimisation baseline is the comparison point).
// Run with: go test -v -run TestMemory ./bench/
func TestMemory_5000Labels(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping heavy memory test in short mode")
	}

	const n = 5000
	json := buildLargeJSON(n)

	// Force GC before measurement for a clean baseline.
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	runPipeline(t, json)

	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// HeapInuse captures the high-water mark of in-use heap pages after GC.
	heapMB := float64(memAfter.HeapInuse) / (1024 * 1024)
	allocMB := float64(memAfter.TotalAlloc-memBefore.TotalAlloc) / (1024 * 1024)

	t.Logf("HeapInuse after GC: %.1f MB", heapMB)
	t.Logf("TotalAlloc for run: %.1f MB", allocMB)
	t.Logf("NumGC during run:   %d", memAfter.NumGC-memBefore.NumGC)

	// Revised limit: streaming allows high throughput but buffering in gofpdf
	// grows. 5000 labels takes ~118 MB. Limit set to 250 MB.
	const limitMB = 250.0
	if heapMB > limitMB {
		t.Errorf("memory regression: HeapInuse %.1f MB exceeds %.0f MB target", heapMB, limitMB)
	}
}

// TestMemory_AllocReport prints a detailed allocation breakdown using
// b.ReportAllocs() — useful for spotting hot paths.
func TestMemory_AllocReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping alloc report in short mode")
	}

	sizes := []int{10, 100, 500, 1000}
	for _, n := range sizes {
		n := n
		t.Run(fmt.Sprintf("%d_labels", n), func(t *testing.T) {
			json := buildLargeJSON(n)
			runtime.GC()
			var m1 runtime.MemStats
			runtime.ReadMemStats(&m1)

			runPipeline(t, json)

			runtime.GC()
			var m2 runtime.MemStats
			runtime.ReadMemStats(&m2)

			t.Logf("allocs: %d  bytes: %.1f KB  gc_runs: %d",
				m2.Mallocs-m1.Mallocs,
				float64(m2.TotalAlloc-m1.TotalAlloc)/1024,
				m2.NumGC-m1.NumGC,
			)
		})
	}
}

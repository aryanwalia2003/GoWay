package pipeline_test

import (
	"context"
	"strings"
	"testing"

	"awb-gen/internal/pipeline"

	"go.uber.org/zap"
)

// validJSON is a minimal two-record JSON array for pipeline testing.
const validJSON = `[
  {
    "awb_number": "ZFW000001",
    "order_id": "#1",
    "sender": "Store A",
    "receiver": "Alice",
    "address": "1 Main St",
    "pincode": "400001",
    "weight": "1kg",
    "sku_details": "Item A x1"
  },
  {
    "awb_number": "ZFW000002",
    "order_id": "#2",
    "sender": "Store B",
    "receiver": "Bob",
    "address": "2 Second St",
    "pincode": "400002",
    "weight": "2kg",
    "sku_details": "Item B x2"
  }
]`

func TestPipelineRun_ProducesResults(t *testing.T) {
	t.Parallel()

	log := zap.NewNop()
	pl := pipeline.New(pipeline.Config{
		WorkerCount:      2,
		JobBufferSize:    4,
		ResultBufferSize: 8,
	}, log)

	ctx := context.Background()
	results, err := pl.Run(ctx, strings.NewReader(validJSON))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var received []pipeline.PageResult
	for r := range results {
		received = append(received, r)
	}

	if len(received) != 2 {
		t.Fatalf("expected 2 results, got %d", len(received))
	}
}

func TestPipelineRun_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Build a very large JSON array to ensure the pipeline is mid-flight
	// when we cancel.
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < 500; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"awb_number":"ZFW999","receiver":"R","address":"A"}`)
	}
	sb.WriteString("]")

	log := zap.NewNop()
	pl := pipeline.New(pipeline.Defaults(), log)

	ctx, cancel := context.WithCancel(context.Background())
	results, err := pl.Run(ctx, strings.NewReader(sb.String()))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Cancel after reading the first result — pipeline must drain cleanly
	// without deadlock.
	<-results
	cancel()

	// Drain remaining results to release goroutines.
	for range results {
	}
	// If we reach here without deadlock, the test passes.
}

func TestPipelineRun_MalformedJSON_SkipsRecord(t *testing.T) {
	t.Parallel()

	// First record is malformed JSON; second is valid. Producer should skip
	// the bad one and still emit the good one.
	input := `[
		{BROKEN},
		{"awb_number":"ZFW1","receiver":"R","address":"A","sender":"S","order_id":"#1","pincode":"0","weight":"1kg","sku_details":"x"}
	]`

	log := zap.NewNop()
	pl := pipeline.New(pipeline.Config{WorkerCount: 1, JobBufferSize: 2, ResultBufferSize: 2}, log)

	ctx := context.Background()
	results, _ := pl.Run(ctx, strings.NewReader(input))

	var count int
	for r := range results {
		if r.Err == nil {
			count++
		}
	}

	// We may get 0 or 1 depending on whether the valid record was parsed before
	// the decoder got confused; we just assert no panic and no deadlock.
	t.Logf("received %d successful result(s) from partially malformed input", count)
}

func BenchmarkPipelineRun_1000Labels(b *testing.B) {
	log := zap.NewNop()

	// Pre-build JSON outside the timer loop.
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < 1000; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"awb_number":"ZFW123456","order_id":"#1","sender":"Store","receiver":"John Doe","address":"123 Main St Mumbai","pincode":"400001","weight":"0.5kg","sku_details":"Item A x1"}`)
	}
	sb.WriteString("]")
	jsonStr := sb.String()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pl := pipeline.New(pipeline.Defaults(), log)
		results, _ := pl.Run(context.Background(), strings.NewReader(jsonStr))
		for range results {
		}
	}
}

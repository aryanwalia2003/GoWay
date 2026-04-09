package assembler

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLaTeXAssembler_ConcurrencyAndTimeout(t *testing.T) {
	assembler := NewLaTeXAssembler("../../tectonic", "../../templates", 2)

	ctx := context.Background()
	var wg sync.WaitGroup

	start := time.Now()

	// Simulate 5 parallel requests on a 2-worker pool
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := assembler.Assemble(ctx, "hello_world", nil)
			// The timeout should forcefully kill these within ~60ms
			if err == nil || !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "killed") && !strings.Contains(err.Error(), "deadline") {
				t.Errorf("Expected context deadline/killed error on local tests taking >60ms, got: %v", err)
			}
		}(i)
	}
	wg.Wait()

	duration := time.Since(start)
	// Even with queueing (5 tasks, 2 workers), total time should be capped by the timeouts.
	// 5 tasks / 2 workers = 3 batches ~ 180ms absolute max.
	if duration > 300*time.Millisecond {
		t.Fatalf("Worker pool failed to enforce strict timeouts, took %v", duration)
	}
}

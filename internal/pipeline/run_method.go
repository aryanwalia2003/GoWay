package pipeline

import (
	"context"
	"io"
	"sync"

	"awb-gen/internal/generator"
)

// Run starts the full SPSC pipeline and returns a result channel.
//
// Goroutine lifecycle:
//   - 1 producer goroutine: streams JSON → jobs channel, closes it on completion
//   - N worker goroutines:  drain jobs channel, emit to results channel
//   - 1 watcher goroutine:  closes results channel once all workers exit
//
// A semaphore of size cfg.MaxConcurrentPDF ensures at most that many
// GenerateLabel calls execute simultaneously, bounding per-worker heap usage
// regardless of how many CPU cores are available.
//
// The caller MUST fully drain the returned channel to avoid goroutine leaks.
// Cancel ctx to abort early; all goroutines respect ctx.Done().
func (p *Pipeline) Run(ctx context.Context, r io.Reader) (<-chan PageResult, error) {
	jobs := make(chan Job, p.cfg.JobBufferSize)
	results := make(chan PageResult, p.cfg.ResultBufferSize)

	// Semaphore: limits simultaneous GenerateLabel calls to MaxConcurrentPDF.
	// Sending acquires a slot; receiving releases it.
	sem := make(chan struct{}, p.cfg.MaxConcurrentPDF)

	// Stage 1: single producer — stream-decodes JSON, never loads full payload
	go p.produce(ctx, r, jobs)

	// Stage 2: worker pool — each worker owns its own isolated generator.
	// The semaphore inside work() caps actual PDF rendering concurrency.
	var wg sync.WaitGroup
	for i := 0; i < p.cfg.WorkerCount; i++ {
		wg.Add(1)
		gen := generator.NewMarotoGenerator(p.encoder)
		go func(g generator.LabelGenerator) {
			defer wg.Done()
			p.work(ctx, jobs, results, g, sem)
		}(gen)
	}

	// Stage 3: single watcher — closes results when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	return results, nil
}
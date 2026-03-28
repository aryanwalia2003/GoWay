package pipeline

import (
	"context"
	"io"
	"sync"

	"awb-gen/internal/generator"
)

// Run starts the AWB generation pipeline.
func (p *Pipeline) Run(ctx context.Context, r io.Reader) (<-chan PageResult, error) {
	jobs := make(chan Job, p.cfg.JobBufferSize)
	results := make(chan PageResult, p.cfg.ResultBufferSize)

	sem := make(chan struct{}, p.cfg.MaxConcurrentPDF)

	go p.produce(ctx, r, jobs)

	// Stage 2: worker pool
	var wg sync.WaitGroup
	for i := 0; i < p.cfg.WorkerCount; i++ {
		wg.Add(1)
		gen := generator.NewMarotoGenerator(p.encoder)
		go func(g generator.LabelGenerator) {
			defer wg.Done()
			p.work(ctx, jobs, results, g, sem)
		}(gen)
	}

	// Stage 3: watcher
	go func() {
		wg.Wait()
		close(results)
	}()

	return results, nil
}
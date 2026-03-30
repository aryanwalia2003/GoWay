package pipeline

import (
	"context"
	"io"
	"sync"

	"awb-gen/internal/generator"
)

// Run starts the pipeline and returns a channel of RenderResults.
// Workers perform barcode encoding and PNG compression in parallel.
// The caller (assembler) draws pages from these results into a single gofpdf doc.
func (p *Pipeline) Run(ctx context.Context, r io.Reader) (<-chan RenderResult, error) {
	jobs := make(chan Job, p.cfg.JobBufferSize)
	results := make(chan RenderResult, p.cfg.ResultBufferSize)

	sem := make(chan struct{}, p.cfg.MaxConcurrentPDF)

	go p.produce(ctx, r, jobs)

	var wg sync.WaitGroup
	for i := 0; i < p.cfg.WorkerCount; i++ {
		wg.Add(1)
		gen := generator.NewMarotoGenerator(p.encoder)
		go func(g generator.LabelGenerator) {
			defer wg.Done()
			p.work(ctx, jobs, results, g, sem)
		}(gen)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results, nil
}
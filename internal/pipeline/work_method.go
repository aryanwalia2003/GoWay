package pipeline

import (
	"context"

	"awb-gen/internal/generator"

	"go.uber.org/zap"
)

// work is the body of each worker goroutine.
func (p *Pipeline) work(
	ctx context.Context,
	jobs <-chan Job,
	results chan<- PageResult,
	gen generator.LabelGenerator,
	sem chan struct{},
) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}

			// Acquire semaphore slot
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}

			pdfBytes, err := gen.GenerateLabel(job.Record)

			// Release semaphore slot
			<-sem

			if err != nil {
				p.log.Error("pipeline/work: label generation failed",
					zap.Int("index", job.Index),
					zap.String("awb_number", job.Record.AWBNumber),
					zap.Error(err))
			}

			result := PageResult{
				Index:    job.Index,
				PDFBytes: pdfBytes,
				Err:      err,
			}

			select {
			case <-ctx.Done():
				return
			case results <- result:
			}
		}
	}
}
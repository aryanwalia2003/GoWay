package pipeline

import (
	"context"

	"awb-gen/internal/generator"

	"go.uber.org/zap"
)

// work is the body of each worker goroutine. It drains the jobs channel until
// it is closed or ctx is cancelled, renders each AWB label via gen, and emits
// the result to the results channel.
//
// Each worker owns its own gen instance — there is no shared mutable state
// between workers. All channel operations are non-blocking via select to ensure
// ctx cancellation is always honoured promptly.
func (p *Pipeline) work(
	ctx context.Context,
	jobs <-chan Job,
	results chan<- PageResult,
	gen generator.LabelGenerator,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				// jobs channel closed — all work is dispatched, exit cleanly
				return
			}

			pdfBytes, err := gen.GenerateLabel(job.Record)
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
package pipeline

import (
	"context"

	"awb-gen/internal/generator"

	"go.uber.org/zap"
)

func (p *Pipeline) work(
	ctx context.Context,
	jobs <-chan Job,
	results chan<- RenderResult,
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

			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}

			pngBytes, err := gen.RenderLabel(job.Record)

			<-sem

			if err != nil {
				p.log.Error("pipeline/work: barcode render failed",
					zap.Int("index", job.Index),
					zap.String("awb_number", job.Record.AWBNumber),
					zap.Error(err),
				)
			}

			result := RenderResult{
				Index:      job.Index,
				Record:     job.Record,
				BarcodePNG: pngBytes,
				Err:        err,
			}

			select {
			case <-ctx.Done():
				return
			case results <- result:
			}
		}
	}
}
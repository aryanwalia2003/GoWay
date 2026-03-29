package pipeline

import (
	"context"
	"github.com/goccy/go-json"
	"io"

	"awb-gen/internal/awb"

	"go.uber.org/zap"
)

// produce is the single producer goroutine. It stream-decodes a JSON array
// from r, validates each record, and sends Jobs to the jobs channel.
func (p *Pipeline) produce(ctx context.Context, r io.Reader, jobs chan<- Job) {
	defer close(jobs)

	dec := json.NewDecoder(r)

	// Consume the opening '[' token of the JSON array.
	if _, err := dec.Token(); err != nil {
		p.log.Error("pipeline/produce: failed to read JSON array open token",
			zap.Error(err))
		return
	}

	index := 0
	for dec.More() {
		// Check for cancellation before blocking on channel send.
		select {
		case <-ctx.Done():
			p.log.Info("pipeline/produce: context cancelled, stopping")
			return
		default:
		}

		var record awb.AWB
		if err := dec.Decode(&record); err != nil {
			p.log.Warn("pipeline/produce: skipping malformed record",
				zap.Int("index", index), zap.Error(err))
			index++
			continue
		}

		if err := record.Validate(); err != nil {
			p.log.Warn("pipeline/produce: skipping invalid record",
				zap.Int("index", index),
				zap.String("awb_number", record.AWBNumber),
				zap.Error(err))
			index++
			continue
		}

		select {
		case <-ctx.Done():
			return
		case jobs <- Job{Index: index, Record: record}:
		}

		index++
	}

	p.log.Info("pipeline/produce: finished reading input", zap.Int("total", index))
}
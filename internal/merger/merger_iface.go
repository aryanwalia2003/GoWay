package merger

import "awb-gen/internal/pipeline"

// Merger is the contract for combining ordered single-page PDFs into one.
type Merger interface {
	// Merge consumes the results channel (already closed by pipeline when done),
	// assembles pages in input order, and returns the final PDF bytes.
	// Returns a fatal error if zero pages were generated.
	Merge(results <-chan pipeline.PageResult) ([]byte, int, error)
}

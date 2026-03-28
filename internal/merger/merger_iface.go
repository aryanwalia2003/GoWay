package merger

import "awb-gen/internal/pipeline"

// Merger is the contract for combining ordered single-page PDFs into one file.
//
// The interface writes directly to outPath on disk instead of returning []byte.
// This eliminates the peak-RSS spike that occurred when the full merged PDF
// was held in a bytes.Buffer before being handed back to the caller.
//
// Returns the number of successfully merged pages, or a fatal error if zero
// pages were generated.
type Merger interface {
	MergeToFile(results <-chan pipeline.PageResult, outPath string) (int, error)
}
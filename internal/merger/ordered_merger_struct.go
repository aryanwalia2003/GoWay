package merger

import "go.uber.org/zap"

// OrderedMerger collects all PageResults, sorts by Index, then concatenates
// using pdfcpu. It runs in a single goroutine — zero write contention.
type OrderedMerger struct {
	log *zap.Logger
	ChunkSize int
}

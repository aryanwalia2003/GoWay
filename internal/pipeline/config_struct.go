package pipeline

import "runtime"

// Config holds tuning parameters for the SPSC pipeline.
// Zero values are replaced with production-safe defaults via Defaults().
type Config struct {
	// WorkerCount is the number of parallel label-rendering goroutines.
	// Defaults to runtime.NumCPU().
	WorkerCount int

	// MaxConcurrentPDF is the maximum number of simultaneous rendering calls.
	// Benchmarked to cap memory usage on high-core-count machines.
	MaxConcurrentPDF int

	// JobBufferSize is the capacity of the job channel.
	JobBufferSize int

	// ResultBufferSize is the capacity of the result channel.
	ResultBufferSize int

	// MergeChunkSize is forwarded to OrderedMerger. It controls how many
	// single-page PDFs are handed to pdfcpu per incremental merge call.
	// Defaults to 500.
	MergeChunkSize int
}

// Defaults returns a Config populated with production-optimal values.
func Defaults() Config {
	workers := runtime.NumCPU()
	maxPDF := workers
	if maxPDF > 8 {
		maxPDF = 8
	}
	return Config{
		WorkerCount:      workers,
		MaxConcurrentPDF: maxPDF,
		JobBufferSize:    workers * 2,
		ResultBufferSize: workers * 4,
		MergeChunkSize:   500,
	}
}
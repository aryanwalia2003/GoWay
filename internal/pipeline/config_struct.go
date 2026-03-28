package pipeline

import "runtime"

// Config holds tuning parameters for the SPSC pipeline.
// Zero values are replaced with production-safe defaults via Defaults().
type Config struct {
	// WorkerCount is the number of parallel label-rendering goroutines.
	// Defaults to runtime.NumCPU().
	WorkerCount int

	// MaxConcurrentPDF is the maximum number of GenerateLabel calls that may
	// execute simultaneously, regardless of WorkerCount.
	//
	// On machines with many CPUs (e.g. 32-core CI runners), WorkerCount alone
	// would spin up 32 MarotoGenerator instances simultaneously, each holding
	// its own render buffer — memory usage would scale linearly with CPU count.
	// MaxConcurrentPDF caps that at a predictable level.
	//
	// Defaults to min(WorkerCount, 8). Tune upward if you have headroom.
	MaxConcurrentPDF int

	// JobBufferSize is the capacity of the job channel (producer → workers).
	// A buffer of WorkerCount*2 balances producer throughput with backpressure.
	JobBufferSize int

	// ResultBufferSize is the capacity of the result channel (workers → merger).
	// Sized to WorkerCount*4 to absorb burst output without blocking workers.
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
package pipeline

import "runtime"

// Config holds tuning parameters for the SPSC pipeline.
// Zero values are replaced with production-safe defaults via Defaults().
type Config struct {
	// WorkerCount is the number of parallel label-rendering goroutines.
	// Defaults to runtime.NumCPU().
	WorkerCount int

	// JobBufferSize is the capacity of the job channel (producer → workers).
	// A buffer of WorkerCount*2 balances producer throughput with backpressure.
	JobBufferSize int

	// ResultBufferSize is the capacity of the result channel (workers → merger).
	// Sized to WorkerCount*4 to absorb burst output without blocking workers.
	ResultBufferSize int
}

// Defaults returns a Config populated with production-optimal values.
func Defaults() Config {
	workers := runtime.NumCPU()
	return Config{
		WorkerCount:      workers,
		JobBufferSize:    workers * 2,
		ResultBufferSize: workers * 4,
	}
}
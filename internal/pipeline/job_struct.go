package pipeline

import "awb-gen/internal/awb"

// Job is the immutable unit of work passed from the producer to the worker pool.
// Index preserves original order for the assembler stage.
type Job struct {
	Index  int
	Record awb.AWB
}
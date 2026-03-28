package pipeline

import "awb-gen/internal/awb"

// Job is the immutable unit of work passed from the JSON reader (producer)
// to the worker pool via the SPSC job channel.
// Index preserves original order for the ordered merge stage.
type Job struct {
	Index  int     // 0-based position in the input stream
	Record awb.AWB // value copy — no pointer, no shared mutable state
}

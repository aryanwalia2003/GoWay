package generator

import "awb-gen/internal/awb"

// LabelGenerator is the contract for rendering a single AWB record into a
// self-contained in-memory PDF byte slice.
//
// Callers (pipeline workers) must never share a LabelGenerator between
// goroutines. Each goroutine must own its own instance to guarantee zero
// shared mutable state and zero write contention.
type LabelGenerator interface {
	GenerateLabel(record awb.AWB) (pdfBytes []byte, err error)
}
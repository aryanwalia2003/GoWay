package pipeline

// PageResult is the output produced by a worker goroutine for a single label.
// It carries the rendered PDF bytes and the original job index so the merger
// can reconstruct document order deterministically without locks.
type PageResult struct {
	Index    int    // matches Job.Index — used for ordered merge
	PDFBytes []byte // complete single-page PDF rendered by the worker
	Err      error  // non-nil if rendering failed; merger skips and logs
}

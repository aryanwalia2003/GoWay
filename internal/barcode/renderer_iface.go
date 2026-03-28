package barcode

// Renderer is the contract for encoding a string into a renderable barcode
// representation. Implementations must be safe for concurrent use — each
// worker goroutine calls Encode independently with no shared mutable state.
type Renderer interface {
	// Encode converts content into a grid of black/white bars expressed as a
	// 2-D bool slice: bars[col][row] == true means "print bar here".
	// Width is the number of bar columns; Height is the logical bar height in
	// PDF points and is set by the caller via dimensions constants.
	Encode(content string) (bars []bool, width int, err error)
}

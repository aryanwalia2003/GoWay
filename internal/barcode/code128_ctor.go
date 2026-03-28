package barcode

// NewCode128Encoder returns a ready-to-use Code128Encoder.
// The returned value is immutable and safe for concurrent use.
func NewCode128Encoder() *Code128Encoder {
	return &Code128Encoder{}
}

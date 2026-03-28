package barcode

// Code128Encoder implements Renderer using the boombuler/barcode library.
// It is stateless; construct once and share freely across goroutines.
type Code128Encoder struct{}

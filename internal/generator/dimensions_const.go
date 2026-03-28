package generator

// Label spatial constants — all millimetre values for maroto's coordinate system.
const (
	MarginMM        = 4.0  // uniform inset on all four edges
	BarcodeHeightMM = 16.0 // height of the barcode image row
)

// Maroto 12-column grid constants.
// Maroto divides every row into 12 equal columns; use named constants
// instead of magic numbers throughout the layout code.
const (
	ColFull = 12 // spans 100% of content width
	ColHalf = 6  // spans 50% of content width
	ColTwo3 = 8  // spans ~67% of content width
	ColOne3 = 4  // spans ~33% of content width
)

// Font size constants in points (float64 — maroto's props.Text.Size type).
const (
	FontSizeTitle  = 9.0
	FontSizeNormal = 7.0
	FontSizeSmall  = 6.0
)

// FontFamily is the registered name for the embedded Roboto font.
// Maroto looks up this string at render time against registered custom fonts.
const FontFamily = "roboto"
package awb

// AWB represents a single Air Waybill label payload decoded from JSON input.
// All fields are value types — no pointers — to minimise GC pressure during
// high-volume streaming decode.
type AWB struct {
	AWBNumber  string `json:"awb_number"`
	OrderID    string `json:"order_id"`
	Sender     string `json:"sender"`
	Receiver   string `json:"receiver"`
	Address    string `json:"address"`
	Pincode    string `json:"pincode"`
	Weight     string `json:"weight"`
	SKUDetails string `json:"sku_details"`
}
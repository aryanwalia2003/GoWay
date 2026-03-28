package main

import (
	"encoding/json"
	"fmt"
	"os"

	"awb-gen/internal/awb"
)

func main() {
	var records []awb.AWB

	for i := 1; i <= 5000; i++ {
		records = append(records, awb.AWB{
			AWBNumber:  fmt.Sprintf("AWB%08d", i),
			OrderID:    fmt.Sprintf("ORD%08d", i),
			Sender:     "Test Sender Inc. 123 Logistics Park",
			Receiver:   fmt.Sprintf("Test Receiver %d", i),
			Address:    "A/19 Example Block, Demo City, India",
			Pincode:    "110001",
			Weight:     "1.5 Kg",
			SKUDetails: "Books (2), Electronics (1)",
		})
	}

	f, err := os.Create("5k_batch.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if err := enc.Encode(records); err != nil {
		panic(err)
	}
	fmt.Println("Generated 5k_batch.json successfully with 5000 records.")
}

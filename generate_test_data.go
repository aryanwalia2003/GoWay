//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"awb-gen/internal/awb"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run generate_test_data.go <count> <filename>")
		return
	}

	countStr := os.Args[1]
	filename := os.Args[2]

	var count int
	fmt.Sscanf(countStr, "%d", &count)

	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Write opening bracket for the array
	f.WriteString("[\n")

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	for i := 1; i <= count; i++ {
		record := awb.AWB{
			AWBNumber:  fmt.Sprintf("AWB%08d", i),
			OrderID:    fmt.Sprintf("ORD%08d", i),
			Sender:     "Test Sender Inc. 123 Logistics Park",
			Receiver:   fmt.Sprintf("Test Receiver %d", i),
			Address:    "A/19 Example Block, Demo City, India",
			Pincode:    "110001",
			Weight:     "1.5 Kg",
			SKUDetails: "Books (2), Electronics (1)",
		}
		
		if err := enc.Encode(record); err != nil {
			panic(err)
		}
		
		if i < count {
			f.WriteString(",")
		}
	}

	// Write closing bracket
	f.WriteString("]\n")

	fmt.Printf("Generated %s successfully with %d records.\n", filename, count)
}

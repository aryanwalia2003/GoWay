package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"awb-gen/internal/assembler"
)

func main() {
	payload := []byte(`{
		"customerName": "sdfgh",
		"address": "hssr layout, None, bangalore - 560001, Tamil Nadu, India",
		"phone": "9876543210",
		"paymentMode": "Online",
		"collectibleAmount": "400",
		"zippeeAwb": "FBXQPY0X54FL7PR",
		"invoiceDate": "2023-01-12 05:30:00",
		"invoiceValue": "400",
		"referenceCode": "cc12-7",
		"brandName": "testing_fabbox",
		"shipperAddress": "Pathardi phata, Nashik, Maharashtra - 560001",
		"returnAddress": "Pathardi phata, Nashik, Maharashtra - 560001",
		"customerSupport": "9599220400",
		"dimensions": "3x3x3",
		"weight": "200"
	}`)

	asm := assembler.NewLaTeXAssembler("./tectonic", "./templates", 1)
	asm.HardCap = 300 * time.Second

	fmt.Println("Generating UC Shipping Label PDF...")
	pdf, err := asm.Assemble(context.Background(), "uc_shipping_label", payload)
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile("uc_label_sample.pdf", pdf, 0644)
	if err != nil {
		fmt.Printf("FAILED to write file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("SUCCESS: uc_label_sample.pdf generated (%d bytes)\n", len(pdf))
}

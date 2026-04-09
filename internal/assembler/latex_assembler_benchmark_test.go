package assembler

import (
	"context"
	"testing"
	"time"
)

func BenchmarkAssembleUCShippingLabel(b *testing.B) {
	payload := []byte(`{
		"customerName": "John Doe",
		"address": "123 Main St, Bangalore",
		"phone": "+91 9876543210",
		"paymentMode": "COD",
		"collectibleAmount": "1500",
		"zippeeAwb": "ZP123456789IN",
		"invoiceDate": "09-Apr-2026",
		"invoiceValue": "1500",
		"referenceCode": "REF-98765",
		"brandName": "UC Brand",
		"shipperAddress": "UC Warehouse, Mumbai",
		"returnAddress": "UC Returns, Pune",
		"customerSupport": "1800-123-4567",
		"dimensions": "20x15x10",
		"weight": "500"
	}`)

	// Using the binary relative to this test file.
	a := NewLaTeXAssembler("../../tectonic", "../../templates", 10)
	a.HardCap = 30 * time.Second
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := a.Assemble(ctx, "uc_shipping_label", payload)
		if err != nil {
			b.Fatalf("expected successful render, got: %v", err)
		}
	}
}

func BenchmarkAssembleUCShippingLabel_Parallel(b *testing.B) {
	payload := []byte(`{
		"customerName": "Parallel User",
		"address": "123 Main St, Bangalore",
		"phone": "+91 9876543210",
		"paymentMode": "COD",
		"collectibleAmount": "1500",
		"zippeeAwb": "ZP123456789PAR",
		"invoiceDate": "09-Apr-2026",
		"invoiceValue": "1500",
		"referenceCode": "REF-PAR-123",
		"brandName": "UC Brand",
		"shipperAddress": "UC Warehouse, Mumbai",
		"returnAddress": "UC Returns, Pune",
		"customerSupport": "1800-123-4567",
		"dimensions": "1x1x1",
		"weight": "1"
	}`)

	a := NewLaTeXAssembler("../../tectonic", "../../templates", 10)
	a.HardCap = 30 * time.Second
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := a.Assemble(ctx, "uc_shipping_label", payload)
			if err != nil {
				b.Fatalf("expected successful render, got: %v", err)
			}
		}
	})
}

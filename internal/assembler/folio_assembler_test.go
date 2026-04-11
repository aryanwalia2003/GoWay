package assembler

import (
	"context"
	"testing"
	"time"
)

func TestFolioAssembler_Caching(t *testing.T) {
	// 1. Setup
	templateDir := "../../templates" // Adjusting for internal/assembler/ package depth
	a := NewFolioAssembler(templateDir)

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

	// 2. First call (cold - caches template)
	start := time.Now()
	pdf1, err := a.Assemble(context.Background(), "uc_shipping_label", payload)
	if err != nil {
		t.Fatalf("first assemble failed: %v", err)
	}
	coldDuration := time.Since(start)
	t.Logf("Cold run duration: %v", coldDuration)

	// 3. Second call (warm - uses cached logo and template)
	start = time.Now()
	pdf2, err := a.Assemble(context.Background(), "uc_shipping_label", payload)
	if err != nil {
		t.Fatalf("second assemble failed: %v", err)
	}
	warmDuration := time.Since(start)
	t.Logf("Warm run duration: %v", warmDuration)

	// Verify PDFs are valid (at least look like PDFs)
	if len(pdf1) < 5 || string(pdf1[:5]) != "%PDF-" {
		t.Errorf("PDF1 invalid header: %q", pdf1[:min(len(pdf1), 5)])
	}
	if len(pdf2) < 5 || string(pdf2[:5]) != "%PDF-" {
		t.Errorf("PDF2 invalid header: %q", pdf2[:min(len(pdf2), 5)])
	}

	// Warm should generally be faster than cold (at least for the I/O part)
	// though Folio rendering itself is the dominant part.
	if warmDuration > coldDuration {
		t.Logf("Note: Warm run (%v) was slower than cold run (%v). This can happen due to GC or OS scheduling, but should be rare for I/O bound tasks.", warmDuration, coldDuration)
	} else {
		t.Logf("Savings: %v", coldDuration-warmDuration)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func BenchmarkAssembleForwardLabel(b *testing.B) {
	templateDir := "../../templates"
	a := NewFolioAssembler(templateDir)

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

	// Warm up cache
	_, _ = a.Assemble(context.Background(), "uc_shipping_label", payload)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := a.Assemble(context.Background(), "uc_shipping_label", payload)
		if err != nil {
			b.Fatalf("assemble failed: %v", err)
		}
	}
}

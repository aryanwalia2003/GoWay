package assembler

import (
	"context"
	"strings"
	"testing"
)

func TestUCShippingLabel_EndToEnd(t *testing.T) {
	// Full payload matching uc_shipping_label.tex field macros
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
		"weight": "500",
		"barcodeZippeeawb": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
		"barcodeRefcode": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	}`)

	a := NewFolioAssembler("../../templates")

	pdfBytes, err := a.Assemble(context.Background(), "uc_shipping_label", payload)
	if err != nil {
		t.Fatalf("expected successful render, got: %v", err)
	}
	t.Logf("Rendered PDF size: %d bytes", len(pdfBytes))

	// Verify we have a valid PDF
	if len(pdfBytes) < 5 || string(pdfBytes[:5]) != "%PDF-" {
		t.Fatalf("expected valid PDF header, got: %q", string(pdfBytes[:min5(pdfBytes)]))
	}
}

func TestUCShippingLabel_MissingFieldsRendersGracefully(t *testing.T) {
	// Only mandatory fields, optional ones missing should render blank
	payload := []byte(`{
		"customerName": "Jane Doe",
		"zippeeAwb": "ZP000000001IN"
	}`)

	a := NewFolioAssembler("../../templates")

	pdfBytes, err := a.Assemble(context.Background(), "uc_shipping_label", payload)
	if err != nil {
		t.Fatalf("missing field should not cause fatal render error: %v", err)
	}

	if len(pdfBytes) < 5 || string(pdfBytes[:5]) != "%PDF-" {
		t.Fatalf("expected valid PDF header, got: %q", pdfBytes)
	}
}

func TestUCShippingLabel_SpecialCharsEscaped(t *testing.T) {
	// Ensure LaTeX special chars in JSON values don't break compilation
	payload := []byte(`{
		"customerName": "O'Brien & Sons",
		"address": "100% Main St, #5",
		"zippeeAwb": "ZP-SPECIAL-IN"
	}`)

	a := NewFolioAssembler("../../templates")

	_, err := a.Assemble(context.Background(), "uc_shipping_label", payload)
	if err != nil && strings.Contains(err.Error(), "fatal") {
		t.Fatalf("special chars caused fatal LaTeX error: %v", err)
	}
}

func min5(b []byte) int {
	if len(b) < 5 {
		return len(b)
	}
	return 5
}

func TestUCShippingLabel_MissingBarcodeBase64_AutoGenerates(t *testing.T) {
	// Full payload missing barcode base64 strings
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

	a := NewFolioAssembler("../../templates")

	pdfBytes, err := a.Assemble(context.Background(), "uc_shipping_label", payload)
	if err != nil {
		t.Fatalf("expected successful render, got: %v", err)
	}
	t.Logf("Rendered PDF size: %d bytes", len(pdfBytes))

	// Verify we have a valid PDF and the length is significantly more than empty
	if len(pdfBytes) < 5 || string(pdfBytes[:5]) != "%PDF-" {
		t.Fatalf("expected valid PDF header, got: %q", string(pdfBytes[:min5(pdfBytes)]))
	}
	if len(pdfBytes) < 1000 {
		t.Fatalf("PDF suspiciously small after barcode generation, size: %d", len(pdfBytes))
	}
}

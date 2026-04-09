package assembler

import (
	"context"
	"testing"
	"time"
)

func TestLaTeXAssembler_AssemblePDF(t *testing.T) {
	// Provide paths to tectonic and templates relative to this test file.
	assembler := NewLaTeXAssembler("../../tectonic", "../../templates", 10)
	assembler.HardCap = 5 * time.Second
	ctx := context.Background()

	// Assemble the hello_world template
	pdfBytes, err := assembler.Assemble(ctx, "hello_world", nil)
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	// Verify we got PDF bytes
	if len(pdfBytes) < 5 || string(pdfBytes[:5]) != "%PDF-" {
		t.Fatalf("Expected PDF header, got: %q", string(pdfBytes[:5]))
	}
}

func TestLaTeXAssembler_AssemblePDFWithData(t *testing.T) {
	assembler := NewLaTeXAssembler("../../tectonic", "../../templates", 10)
	assembler.HardCap = 5 * time.Second
	ctx := context.Background()

	payload := []byte(`{"message": "Dynamic Content"}`)

	pdfBytes, err := assembler.Assemble(ctx, "hello_world_dynamic", payload)
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	if len(pdfBytes) < 5 || string(pdfBytes[:5]) != "%PDF-" {
		t.Fatalf("Expected PDF header, got: %q", string(pdfBytes[:5]))
	}
}

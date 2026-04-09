package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockAssembler lets tests control rendering output without invoking tectonic.
type mockAssembler struct {
	pdfBytes []byte
	err      error
}

func (m *mockAssembler) Assemble(_ context.Context, _ string, _ []byte) ([]byte, error) {
	return m.pdfBytes, m.err
}

func TestRenderForwardShippingLabel_Success(t *testing.T) {
	mock := &mockAssembler{pdfBytes: []byte("%PDF-1.5 success")}
	handler := NewLaTeXHandler(mock)

	body := bytes.NewBufferString(`{"template_id":"hello_world","data":{"awb":"123"}}`)
	req := httptest.NewRequest(http.MethodPost, "/render/forward-shipping-label", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("expected Content-Type application/pdf, got %s", ct)
	}
	buf, _ := io.ReadAll(resp.Body)
	if string(buf) != "%PDF-1.5 success" {
		t.Fatalf("unexpected body: %s", string(buf))
	}
}

func TestRenderForwardShippingLabel_MissingTemplateID(t *testing.T) {
	mock := &mockAssembler{}
	handler := NewLaTeXHandler(mock)

	body := bytes.NewBufferString(`{"data":{"awb":"123"}}`)
	req := httptest.NewRequest(http.MethodPost, "/render/forward-shipping-label", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRenderForwardShippingLabel_TemplateNotFound(t *testing.T) {
	mock := &mockAssembler{err: ErrTemplateNotFound}
	handler := NewLaTeXHandler(mock)

	body := bytes.NewBufferString(`{"template_id":"ghost","data":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/render/forward-shipping-label", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRenderForwardShippingLabel_CompilationTimeout(t *testing.T) {
	mock := &mockAssembler{err: ErrRenderTimeout}
	handler := NewLaTeXHandler(mock)

	body := bytes.NewBufferString(`{"template_id":"slow","data":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/render/forward-shipping-label", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusRequestTimeout {
		t.Fatalf("expected 408, got %d", w.Code)
	}
}

func TestRenderForwardShippingLabel_EmptyBody(t *testing.T) {
	mock := &mockAssembler{}
	handler := NewLaTeXHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/render/forward-shipping-label", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", w.Code)
	}
}

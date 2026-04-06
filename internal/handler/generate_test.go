package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_ValidPayloadAccepted(t *testing.T) {
	body := `[{"awb_number": "123", "order_id": "456", "sender": "S", "receiver": "R", "address": "A", "pincode": "P", "weight": "W", "sku_details": "SKU"}]`
	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleGenerate(rr, req)

	// Since we haven't implemented pipeline, it should accept the payload and return 200 or at least not 4xx
	if rr.Code >= 400 {
		t.Errorf("Expected non-4xx status, got %v", rr.Code)
	}
}

func TestHandler_MalformedJSON(t *testing.T) {
	body := `{broken_json:`
	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleGenerate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected %v, got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_MissingRequiredFields(t *testing.T) {
	body := `[{}]`
	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleGenerate(rr, req)

	if rr.Code != http.StatusUnprocessableEntity && rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 422 or 400 for missing fields, got %v", rr.Code)
	}
}

func TestHandler_EmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/generate", nil)
	rr := httptest.NewRecorder()

	HandleGenerate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected %v, got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_DeepNestedJSON(t *testing.T) {
	body := strings.Repeat(`{"a": `, 1000) + "{}" + strings.Repeat(`}`, 1000)
	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleGenerate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected %v, got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestHandler_OversizedPayload(t *testing.T) {
	body := strings.Repeat(`{"awb_number": "x"},`, 200000)
	body = "[" + body[:len(body)-1] + "]" // roughly 4MB

	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleGenerate(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge && rr.Code != http.StatusBadRequest {
		t.Logf("Got response body: %s", rr.Body.String())
		t.Errorf("Expected 413 or 400 for oversized payload, got %v", rr.Code)
	}
}

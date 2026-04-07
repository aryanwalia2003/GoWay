package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"awb-gen/internal/logger"
)

func TestMain(m *testing.M) {
	logger.InitTestLogger(os.Stdout)
	os.Exit(m.Run())
}

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

func TestGenerate_ValidBatchReturnsPDF(t *testing.T) {
	body := `[{"awb_number": "123", "order_id": "O1", "sender": "S1", "receiver": "R1", "address": "A1", "pincode": "P1", "weight": "W1", "sku_details": "SK1"}]`
	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleGenerate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %v", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/pdf" {
		t.Errorf("Expected Content-Type application/pdf, got %s", contentType)
	}

	if !bytes.HasPrefix(rr.Body.Bytes(), []byte("%PDF-")) {
		t.Error("Expected response body to be a PDF")
	}
}

func TestGenerate_ResponseIsBuffered(t *testing.T) {
	body := `[{"awb_number": "123", "order_id": "O1", "sender": "S1", "receiver": "R1", "address": "A1", "pincode": "P1", "weight": "W1", "sku_details": "SK1"}]`
	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleGenerate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %v", rr.Code)
	}

	if rr.Header().Get("Transfer-Encoding") == "chunked" {
		t.Error("Expected response NOT to be chunked")
	}

	if rr.Header().Get("Content-Length") == "" {
		t.Error("Expected Content-Length header to be set (fully buffered)")
	}
}

func TestGenerate_PartialSuccessOmitsFailedLabels(t *testing.T) {
	body := `[
		{"awb_number": "123", "order_id": "O1", "sender": "S1", "receiver": "R1", "address": "A1", "pincode": "P1", "weight": "W1", "sku_details": "SK1"},
		{"awb_number": "456", "order_id": "O2", "sender": "S2", "receiver": "R2", "pincode": "P2", "weight": "W2", "sku_details": "SK2"}
	]`
	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleGenerate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for partial success, got %v", rr.Code)
	}

	if rr.Header().Get("X-Failed-Count") != "1" {
		t.Errorf("Expected X-Failed-Count to be 1, got %v", rr.Header().Get("X-Failed-Count"))
	}
}

func TestGenerate_SingleRecordBatch(t *testing.T) {
	body := `[{"awb_number": "123", "order_id": "O1", "sender": "S1", "receiver": "R1", "address": "A1", "pincode": "P1", "weight": "W1", "sku_details": "SK1"}]`
	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleGenerate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %v", rr.Code)
	}
}

func TestGenerate_EmptyBatch(t *testing.T) {
	body := `[]`
	req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	HandleGenerate(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected 400 or 422 for empty array, got %v", rr.Code)
	}
}

package logger_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"awb-gen/internal/logger"

	"github.com/google/uuid"
)

// TestLogger_FieldContract verifies that LogRequest emits the required fields.
func TestLogger_FieldContract(t *testing.T) {
	var buf bytes.Buffer
	logger.InitTestLogger(&buf)

	traceID := uuid.New().String()
	logger.LogRequest(
		traceID,
		"POST",
		"/generate",
		200,
		120,
		[]string{"ZPE999"},
	)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to unmarshal log JSON: %v. Output: %s", err, buf.String())
	}

	requiredKeys := []string{
		"level", "ts", "trace_id", "source", "msg",
		"method", "path", "status_code", "duration_ms", "failed_awbs",
	}

	for _, key := range requiredKeys {
		if _, ok := logEntry[key]; !ok {
			t.Errorf("missing required log key: %s", key)
		}
	}

	if source, ok := logEntry["source"].(string); !ok || source != "goway" {
		t.Errorf("expected source to be 'goway', got %v", logEntry["source"])
	}
}

// TestLogger_TraceIdUnique ensures that generating trace IDs works and they are unique.
func TestLogger_TraceIdUnique(t *testing.T) {
	id1 := logger.GenerateTraceID()
	id2 := logger.GenerateTraceID()

	if id1 == "" || id2 == "" {
		t.Errorf("GenerateTraceID returned empty string")
	}

	if id1 == id2 {
		t.Errorf("GenerateTraceID returned identical IDs: %s", id1)
	}

	if _, err := uuid.Parse(id1); err != nil {
		t.Errorf("GenerateTraceID did not return a valid UUID: %v", err)
	}
}

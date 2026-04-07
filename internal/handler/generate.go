package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"awb-gen/internal/awb"
)

const maxPayloadSize = 2 << 20 // 2MB

// HandleGenerate processes AWB batches using streaming.
func HandleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil || r.ContentLength == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadSize)

	if err := processStream(w, r.Body); err != nil {
		handleError(w, err)
	}
}

// processStream orchestrates the JSON array decoding.
func processStream(w http.ResponseWriter, body io.ReadCloser) error {
	dec := json.NewDecoder(body)
	if err := ensureArrayStart(dec); err != nil {
		return err
	}
	return streamRecords(w, dec)
}

// ensureArrayStart expects the first token to be '['.
func ensureArrayStart(dec *json.Decoder) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '[' {
		return errors.New("expected JSON array")
	}
	return nil
}

// streamRecords decodes and validates objects until the array ends.
func streamRecords(w http.ResponseWriter, dec *json.Decoder) error {
	count, failed := 0, 0
	for dec.More() {
		if err := decodeAndValidate(dec); err != nil {
			var valErr *validationError
			if errors.As(err, &valErr) {
				failed++
				count++
				continue
			}
			return err
		}
		count++
	}

	if count == 0 {
		return errors.New("empty array")
	}
	if count == failed {
		return &validationError{errors.New("all records failed validation")}
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

// decodeAndValidate processes a single AWB record.
func decodeAndValidate(dec *json.Decoder) error {
	var a awb.AWB
	if err := dec.Decode(&a); err != nil {
		return err
	}
	if err := a.Validate(); err != nil {
		return &validationError{err}
	}
	return nil
}

// validationError wraps domain validation errors to separate from decode errors.
type validationError struct{ err error }

func (e *validationError) Error() string { return e.err.Error() }

// handleError writes the correct HTTP status based on the error type.
func handleError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest

	var valErr *validationError
	if errors.As(err, &valErr) || err.Error() == "empty array" {
		status = http.StatusUnprocessableEntity
	} else if err.Error() == "http: request body too large" {
		status = http.StatusRequestEntityTooLarge
	}

	http.Error(w, err.Error(), status)
}

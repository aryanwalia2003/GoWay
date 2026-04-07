package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"awb-gen/internal/assembler"
	"awb-gen/internal/assets"
	"awb-gen/internal/awb"
	"awb-gen/internal/logger"
	"awb-gen/internal/pipeline"
)

const maxPayloadSize = 2 << 20 // 2MB

// HandleGenerate processes AWB batches using streaming but buffers the PDF.
func HandleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil || r.ContentLength == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadSize)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		handleError(w, err)
		return
	}

	_, failed, err := validatePayload(bodyBytes)
	if err != nil {
		handleError(w, err)
		return
	}

	if failed > 0 {
		w.Header().Set("X-Failed-Count", strconv.Itoa(failed))
	}

	pl := pipeline.New(pipeline.Defaults(), logger.Log)
	results, err := pl.Run(r.Context(), bytes.NewReader(bodyBytes))
	if err != nil {
		handleError(w, err)
		return
	}

	asm := assembler.New(logger.Log, assets.RobotoRegular, assets.RobotoBold)

	var buf bytes.Buffer
	drawn, err := asm.AssembleToWriter(results, &buf)
	if err != nil {
		if drawn == 0 {
			handleError(w, &validationError{errors.New("all records failed validation")})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

func validatePayload(body []byte) (int, int, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := ensureArrayStart(dec); err != nil {
		return 0, 0, err
	}
	return streamRecordsCount(dec)
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

func streamRecordsCount(dec *json.Decoder) (int, int, error) {
	count, failed := 0, 0
	for dec.More() {
		if err := decodeAndValidate(dec); err != nil {
			var valErr *validationError
			if errors.As(err, &valErr) {
				failed++
				count++
				continue
			}
			return 0, 0, err
		}
		count++
	}

	if count == 0 {
		return 0, 0, errors.New("empty array")
	}
	if count == failed {
		return count, failed, &validationError{errors.New("all records failed validation")}
	}
	return count, failed, nil
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

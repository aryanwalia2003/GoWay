package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

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

	_, failedAWBs, err := validatePayload(bodyBytes)
	failed := len(failedAWBs)
	if failed > 0 {
		w.Header().Set("X-Failed-Count", strconv.Itoa(failed))
		if header := buildFailedAWBsHeader(failedAWBs); header != "" {
			w.Header().Set("X-Failed-AWBs", header)
		}
	}

	if err != nil {
		handleError(w, err)
		return
	}

	pl := pipeline.New(pipeline.Defaults(), logger.Log)
	results, err := pl.Run(r.Context(), bytes.NewReader(bodyBytes))
	if err != nil {
		handleError(w, err)
		return
	}

	asm := assembler.New(logger.Log, assets.RobotoRegular, assets.RobotoBold)

	var buf bytes.Buffer
	drawn, renderFailedAWBs, err := asm.AssembleToWriter(results, &buf)

	failedAWBs = append(failedAWBs, renderFailedAWBs...)
	failed = len(failedAWBs)

	if failed > 0 {
		w.Header().Set("X-Failed-Count", strconv.Itoa(failed))
		if header := buildFailedAWBsHeader(failedAWBs); header != "" {
			w.Header().Set("X-Failed-AWBs", header)
		}
	}

	if err != nil {
		if drawn == 0 {
			handleError(w, &validationError{errors.New("all records failed rendering")})
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

func buildFailedAWBsHeader(failed []string) string {
	if len(failed) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, id := range failed {
		part := id
		if i > 0 {
			part = "," + id
		}
		if sb.Len()+len(part) > 4096 {
			break
		}
		sb.WriteString(part)
	}
	return sb.String()
}

func validatePayload(body []byte) (int, []string, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := ensureArrayStart(dec); err != nil {
		return 0, nil, err
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

func streamRecordsCount(dec *json.Decoder) (int, []string, error) {
	count := 0
	var failedAWBs []string

	for dec.More() {
		awbNum, err := decodeAndValidate(dec)
		if err != nil {
			var valErr *validationError
			if errors.As(err, &valErr) {
				failedAWBs = append(failedAWBs, awbNum)
				count++
				continue
			}
			return 0, nil, err
		}
		count++
	}

	if count == 0 {
		return 0, nil, errors.New("empty array")
	}
	failed := len(failedAWBs)
	if count == failed {
		return count, failedAWBs, &validationError{errors.New("all records failed validation")}
	}
	return count, failedAWBs, nil
}

// decodeAndValidate processes a single AWB record. Returns the AWBNumber and any error.
func decodeAndValidate(dec *json.Decoder) (string, error) {
	var a awb.AWB
	if err := dec.Decode(&a); err != nil {
		return "", err
	}
	if err := a.Validate(); err != nil {
		return a.AWBNumber, &validationError{err}
	}
	return a.AWBNumber, nil
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

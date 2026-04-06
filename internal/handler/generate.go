package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"awb-gen/internal/awb"
)

const maxPayloadSize = 2 << 20 // 2MB

// HandleGenerate validates and processes incoming AWB batches.
func HandleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil || r.ContentLength == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadSize)

	var payload []awb.AWB
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "http: request body too large" {
			status = http.StatusRequestEntityTooLarge
		}
		http.Error(w, err.Error(), status)
		return
	}

	if err := validatePayload(payload); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func validatePayload(payload []awb.AWB) error {
	if len(payload) == 0 {
		return errors.New("empty array")
	}
	for i := range payload {
		if err := payload[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}

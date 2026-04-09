package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// Sentinel errors used by the handler and assembler interface.
var (
	ErrTemplateNotFound = errors.New("template not found")
	ErrRenderTimeout    = errors.New("render timeout")
)

// LaTeXRenderAssembler is the interface the handler depends on.
type LaTeXRenderAssembler interface {
	Assemble(ctx context.Context, templateID string, payload []byte) ([]byte, error)
}

// LaTeXHandler handles POST /render/forward-shipping-label.
type LaTeXHandler struct {
	assembler LaTeXRenderAssembler
}

// NewLaTeXHandler creates the handler with injected assembler.
func NewLaTeXHandler(a LaTeXRenderAssembler) *LaTeXHandler {
	return &LaTeXHandler{assembler: a}
}

// ServeHTTP handles the render request.
func (h *LaTeXHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	req, err := decodeRenderRequest(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.render(w, r, req)
}

func decodeRenderRequest(body io.Reader) (*renderRequest, error) {
	var req renderRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		return nil, err
	}
	if req.TemplateID == "" {
		return nil, errors.New("template_id is required")
	}
	return &req, nil
}

func (h *LaTeXHandler) render(w http.ResponseWriter, r *http.Request, req *renderRequest) {
	payload, _ := json.Marshal(req.Data)

	pdfBytes, err := h.assembler.Assemble(r.Context(), req.TemplateID, payload)
	if err != nil {
		writeRenderError(w, err)
		return
	}

	writePDFResponse(w, pdfBytes)
}

func writeRenderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrTemplateNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, ErrRenderTimeout):
		http.Error(w, err.Error(), http.StatusRequestTimeout)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writePDFResponse(w http.ResponseWriter, pdf []byte) {
	w.Header().Set("Content-Type", "application/pdf")
	w.WriteHeader(http.StatusOK)
	w.Write(pdf)
}

type renderRequest struct {
	TemplateID string                 `json:"template_id"`
	Data       map[string]interface{} `json:"data"`
}

package handler

import (
	"fmt"
	"net/http"
)

// ReadinessChecker is an interface for components that can report their readiness.
type ReadinessChecker interface {
	Ready() error
}

// HealthHandler contains dependencies for health checks.
type HealthHandler struct {
	checker ReadinessChecker
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(c ReadinessChecker) *HealthHandler {
	return &HealthHandler{checker: c}
}

// HandleHealthz is a general health check endpoint.
func (h *HealthHandler) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// HandleLivez is the liveness probe endpoint.
func (h *HealthHandler) HandleLivez(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// HandleReadyz is the readiness probe endpoint.
func (h *HealthHandler) HandleReadyz(w http.ResponseWriter, r *http.Request) {
	if h.checker != nil {
		if err := h.checker.Ready(); err != nil {
			http.Error(w, fmt.Sprintf("NOT READY: %v", err), http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

package pipeline

import (
	"awb-gen/internal/barcode"

	"go.uber.org/zap"
)

// Pipeline owns the full SPSC processing graph:
//
//	JSON reader → job channel → worker pool → result channel → (caller drains)
//
// Construct once with New(), call Run() once, drain the returned channel.
// Not reusable across calls.
type Pipeline struct {
	cfg     Config
	log     *zap.Logger
	encoder barcode.Renderer
}
package pipeline

import (
	"awb-gen/internal/barcode"

	"go.uber.org/zap"
)

// New constructs a Pipeline with the given config and logger.
// If cfg has zero values, Defaults() is applied automatically.
// The barcode encoder is stateless and shared across all workers as a
// read-only dependency — safe with no synchronisation.
func New(cfg Config, log *zap.Logger) *Pipeline {
	if cfg.WorkerCount == 0 {
		cfg = Defaults()
	}
	return &Pipeline{
		cfg:     cfg,
		log:     log,
		encoder: barcode.NewCode128Encoder(),
	}
}
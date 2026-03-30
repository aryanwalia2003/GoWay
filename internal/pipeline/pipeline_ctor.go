package pipeline

import (
	"awb-gen/internal/barcode"

	"go.uber.org/zap"
)

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
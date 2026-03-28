package merger

import "go.uber.org/zap"

func NewOrderedMerger(log *zap.Logger) *OrderedMerger {
	return &OrderedMerger{log: log}
}

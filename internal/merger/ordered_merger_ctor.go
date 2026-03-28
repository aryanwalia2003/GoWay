package merger

import "go.uber.org/zap"

const defaultChunkSize = 500

func NewOrderedMerger(log *zap.Logger, chunkSize int) *OrderedMerger {
	if chunkSize <= 0 {
		chunkSize=defaultChunkSize
	}
		
	return &OrderedMerger{log: log, ChunkSize: chunkSize}
}

package assembler

import "go.uber.org/zap"

// New returns a GofpdfAssembler. Font bytes must be the same copies passed
// to the workers — they are read-only after this point.
func New(log *zap.Logger, regularFont, boldFont []byte) *GofpdfAssembler {
	return &GofpdfAssembler{
		log:         log,
		regularFont: regularFont,
		boldFont:    boldFont,
	}
}

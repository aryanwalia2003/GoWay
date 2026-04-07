package assembler

import (
	"awb-gen/internal/pipeline"
	"io"
)

// Assembler consumes an ordered stream of RenderResults and writes a single
// structurally valid PDF to outPath. It owns the gofpdf instance for the
// entire batch — fonts are registered once, pages are added sequentially.
type Assembler interface {
	AssembleToFile(results <-chan pipeline.RenderResult, outPath string) (int, []string, error)
	AssembleToWriter(results <-chan pipeline.RenderResult, w io.Writer) (int, []string, error)
}

package assembler

import (
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// mergeFiles combines validPDFPaths into outPath using pdfcpu.
// Each input file must be a structurally valid standalone PDF —
// pdfcpu only rebuilds xref tables, it does not re-render content.
func mergeFiles(validPDFPaths []string, outPath string) error {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	return api.MergeCreateFile(validPDFPaths, outPath, false, conf)
}

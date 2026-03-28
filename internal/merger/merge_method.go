package merger

import (
	"bytes"
	"fmt"
	"io"
	"sort"

	"awb-gen/internal/pipeline"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"go.uber.org/zap"
)

// Merge drains the results channel (closed by the pipeline once all workers
// finish), restores original input order via index sort, then concatenates all
// single-page PDFs into one document using pdfcpu — entirely in memory.
//
// Soft per-page errors are logged and skipped; the batch continues.
// Returns a fatal error only when zero pages were successfully rendered.
func (m *OrderedMerger) Merge(results <-chan pipeline.PageResult) ([]byte, int, error) {
	collected := m.drain(results)

	if len(collected) == 0 {
		return nil, 0, fmt.Errorf("merger: no successfully rendered pages available")
	}

	// Restore original JSON input order in O(n log n).
	sort.Slice(collected, func(i, j int) bool {
		return collected[i].Index < collected[j].Index
	})

	merged, err := m.concat(collected)
	if err != nil {
		return nil, 0, err
	}

	m.log.Info("merger: complete", zap.Int("pages", len(collected)))
	return merged, len(collected), nil
}

// drain reads every PageResult from the channel until it is closed,
// logging and discarding failed pages.
func (m *OrderedMerger) drain(results <-chan pipeline.PageResult) []pipeline.PageResult {
	var out []pipeline.PageResult
	for r := range results {
		if r.Err != nil {
			m.log.Warn("merger: skipping failed page",
				zap.Int("index", r.Index),
				zap.Error(r.Err))
			continue
		}
		out = append(out, r)
	}
	return out
}

// concat merges the ordered PDF byte slices into a single document using
// pdfcpu's in-memory merge API. No temp files are written.
func (m *OrderedMerger) concat(pages []pipeline.PageResult) ([]byte, error) {
	readers := make([]io.ReadSeeker, len(pages))
	for i, p := range pages {
		readers[i] = bytes.NewReader(p.PDFBytes)
	}

	var out bytes.Buffer
	// Pre-allocate: ~10 KB per label is a conservative upper bound that avoids
	// the first few reallocs on large batches without wasting memory on small ones.
	out.Grow(len(pages) * 10 * 1024)

	// pdfcpu v0.7.0: NewDefaultConfiguration() returns a *model.Configuration
	// with sensible defaults. We only override OptimizeResourceDicts to speed up
	// the merge step — we skip PDF/A and encryption checks.
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	if err := api.MergeRaw(readers, &out, false, conf); err != nil {
		return nil, fmt.Errorf("merger: pdfcpu.MergeRaw: %w", err)
	}
	return out.Bytes(), nil
}
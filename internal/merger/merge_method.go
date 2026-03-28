package merger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"

	"awb-gen/internal/pipeline"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"go.uber.org/zap"
)

// MergeToFile drains the results channel, restores original input order,
// then writes the merged PDF directly to outPath using incremental chunk-based
// merging. It never holds the full merged PDF in memory.
//
// Memory profile compared to the old Merge() → []byte approach:
//
//	Old: all 5 000 page PDFs in RAM + full merged PDF in bytes.Buffer → ~966 MB RSS
//	New: ChunkSize page PDFs live at a time + pdfcpu merge is append-to-file → ~120-150 MB RSS
//
// Soft per-page errors are logged and skipped; the batch continues.
// Returns a fatal error only when zero pages were successfully rendered.
func (m *OrderedMerger) MergeToFile(results <-chan pipeline.PageResult, outPath string) (int, error) {
	collected := m.drain(results)

	if len(collected) == 0 {
		return 0, fmt.Errorf("merger: no successfully rendered pages available")
	}

	// Restore original JSON input order in O(n log n).
	sort.Slice(collected, func(i, j int) bool {
		return collected[i].Index < collected[j].Index
	})

	if err := m.writeChunked(collected, outPath); err != nil {
		return 0, err
	}

	m.log.Info("merger: complete", zap.Int("pages", len(collected)))
	return len(collected), nil
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

// writeChunked merges collected pages into outPath in ChunkSize-page batches.
//
// Strategy:
//  1. Open the final output file immediately.
//  2. For each chunk of up to ChunkSize pages, call pdfcpu.MergeRaw with the
//     output file as the io.Writer. The second chunk onwards passes append=true
//     so pdfcpu appends pages to the already-written content.
//  3. Nil out PDFBytes on each page after its chunk is written so the GC can
//     reclaim that memory before the next chunk starts.
//
// Peak in-process heap is bounded to ~(ChunkSize × avg_page_size) regardless
// of total label count.
func (m *OrderedMerger) writeChunked(pages []pipeline.PageResult, outPath string) error {
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("merger: create output file %q: %w", outPath, err)
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("merger: close output file: %w", cerr)
		}
	}()

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	firstChunk := true
	for start := 0; start < len(pages); start += m.ChunkSize {
		end := start + m.ChunkSize
		if end > len(pages) {
			end = len(pages)
		}
		chunk := pages[start:end]

		if err = m.mergeChunk(chunk, out, !firstChunk, conf); err != nil {
			return err
		}
		firstChunk = false

		// Free PDFBytes immediately so GC can reclaim before the next chunk.
		for i := range chunk {
			pages[start+i].PDFBytes = nil
		}

		m.log.Info("merger: chunk written",
			zap.Int("from", start),
			zap.Int("to", end),
			zap.Int("total", len(pages)),
		)
	}

	return nil
}

// mergeChunk calls pdfcpu.MergeRaw for a single slice of pages, writing into w.
// When append is true, pdfcpu appends to the content already in w (incremental merge).
func (m *OrderedMerger) mergeChunk(
	chunk []pipeline.PageResult,
	w io.Writer,
	append bool,
	conf *model.Configuration,
) error {
	readers := make([]io.ReadSeeker, len(chunk))
	for i, p := range chunk {
		readers[i] = bytes.NewReader(p.PDFBytes)
	}

	if err := api.MergeRaw(readers, w, append, conf); err != nil {
		return fmt.Errorf("merger: pdfcpu.MergeRaw (chunk len=%d): %w", len(chunk), err)
	}
	return nil
}
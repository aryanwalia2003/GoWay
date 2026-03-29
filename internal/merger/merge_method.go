package merger

import (
	"bytes"
	"container/heap"
	"fmt"
	"io"
	"os"

	"awb-gen/internal/pipeline"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"go.uber.org/zap"
)

// MergeToFile consumes results as they arrive from the pipeline, maintaining
// order via a min-heap on Index, and flushes to disk in ChunkSize-page
// batches — all while workers are still running.
//
// Soft per-page errors are logged and skipped; the batch continues.
// Returns a fatal error only when zero pages were successfully rendered.
func (m *OrderedMerger) MergeToFile(results <-chan pipeline.PageResult, outPath string) (int, error) {
	out, err := os.Create(outPath)
	if err != nil {
		return 0, fmt.Errorf("merger: create output file %q: %w", outPath, err)
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("merger: close output file: %w", cerr)
		}
	}()

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	w := &windowedWriter{
		log:        m.log,
		out:        out,
		conf:       conf,
		chunkSize:  m.ChunkSize,
		h:          &resultHeap{},
		nextIndex:  0,
		firstFlush: true,
	}
	heap.Init(w.h)

	// Consume the results channel as it arrives — workers are still running.
	// Each result is pushed onto the min-heap, then we drain any contiguous
	// in-order run from the heap top into the chunk buffer.
	for r := range results {
		if r.Err != nil {
			m.log.Warn("merger: skipping failed page",
				zap.Int("index", r.Index),
				zap.Error(r.Err))
			// Push a nil-bytes sentinel so drainReady can advance past this
			// index without stalling on a gap that will never be filled.
			heap.Push(w.h, pipeline.PageResult{Index: r.Index, PDFBytes: nil, Err: r.Err})
		} else {
			heap.Push(w.h, r)
		}

		if err = w.drainReady(); err != nil {
			return 0, err
		}
	}

	// Channel closed — all workers done. Drain whatever remains in the heap.
	// These are pages that arrived out-of-order and were held back waiting for
	// predecessors that either arrived later or were errors. Now we can emit all.
	if err = w.flushAll(); err != nil {
		return 0, err
	}

	// Flush any partial chunk that did not reach ChunkSize.
	if err = w.flushChunk(); err != nil {
		return 0, err
	}

	if w.totalFlushed == 0 {
		return 0, fmt.Errorf("merger: no successfully rendered pages available")
	}

	m.log.Info("merger: complete", zap.Int("pages", w.totalFlushed))
	return w.totalFlushed, nil
}

// ── windowedWriter ────────────────────────────────────────────────────────────

// windowedWriter holds all streaming merge state: the min-heap for in-order
// delivery, the current chunk accumulator, and the output file handle.
// It is used only from within MergeToFile — never shared across goroutines.
type windowedWriter struct {
	log          *zap.Logger
	out          *os.File
	conf         *model.Configuration
	chunkSize    int
	h            *resultHeap
	chunk        []pipeline.PageResult // current accumulation buffer
	nextIndex    int                   // next sequential index expected
	firstFlush   bool
	totalFlushed int
}

// drainReady pops all heap entries whose Index == nextIndex, appends
// them to the chunk buffer, and flushes to disk whenever ChunkSize
// pages have accumulated.
func (w *windowedWriter) drainReady() error {
	for w.h.Len() > 0 && (*w.h)[0].Index == w.nextIndex {
		r := heap.Pop(w.h).(pipeline.PageResult)
		w.nextIndex++

		if r.PDFBytes == nil {
			// Sentinel for a skipped/errored page — advance the counter, skip buffering.
			continue
		}

		w.chunk = append(w.chunk, r)

		if len(w.chunk) >= w.chunkSize {
			if err := w.flushChunk(); err != nil {
				return err
			}
		}
	}
	return nil
}

// flushAll drains the entire heap regardless of ordering gaps. Called once
// after the results channel closes — at that point no new pages will arrive
// so any gap is permanent (an error page whose sentinel is already in the heap).
func (w *windowedWriter) flushAll() error {
	for w.h.Len() > 0 {
		r := heap.Pop(w.h).(pipeline.PageResult)
		if r.PDFBytes == nil {
			continue // error sentinel — skip
		}
		w.chunk = append(w.chunk, r)
		if len(w.chunk) >= w.chunkSize {
			if err := w.flushChunk(); err != nil {
				return err
			}
		}
	}
	return nil
}

// flushChunk writes the current chunk accumulator to the output file via
// pdfcpu, then immediately nils every PDFBytes pointer so the GC can reclaim
// that memory before the next chunk starts accumulating.
//
// The backing array of w.chunk is reused (w.chunk = w.chunk[:0]) to avoid
// reallocating a new slice header on every flush — one allocation for the
// whole run after the first chunk.
func (w *windowedWriter) flushChunk() error {
	if len(w.chunk) == 0 {
		return nil
	}

	readers := make([]io.ReadSeeker, len(w.chunk))
	for i, p := range w.chunk {
		readers[i] = bytes.NewReader(p.PDFBytes)
	}

	appendMode := !w.firstFlush
	if err := api.MergeRaw(readers, w.out, appendMode, w.conf); err != nil {
		return fmt.Errorf("merger: pdfcpu.MergeRaw (chunk len=%d): %w", len(w.chunk), err)
	}
	w.firstFlush = false

	w.log.Info("merger: chunk flushed",
		zap.Int("pages_in_chunk", len(w.chunk)),
		zap.Int("total_flushed", w.totalFlushed+len(w.chunk)),
	)

	w.totalFlushed += len(w.chunk)

	// Nil PDFBytes so GC can reclaim page memory immediately.
	for i := range w.chunk {
		w.chunk[i].PDFBytes = nil
	}
	w.chunk = w.chunk[:0] // reset length, keep backing array

	return nil
}

// ── resultHeap ───────────────────────────────────────────────────────────────

// resultHeap is a min-heap of PageResult values ordered by Index.
// It implements heap.Interface for container/heap management.
// The heap holds pages that arrived out-of-order until their predecessors arrive.
type resultHeap []pipeline.PageResult

func (h resultHeap) Len() int           { return len(h) }
func (h resultHeap) Less(i, j int) bool { return h[i].Index < h[j].Index }
func (h resultHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *resultHeap) Push(x any) {
	*h = append(*h, x.(pipeline.PageResult))
}

func (h *resultHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = pipeline.PageResult{} // zero slot to release PDFBytes to GC
	*h = old[:n-1]
	return x
}
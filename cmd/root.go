package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"awb-gen/internal/logger"
	"awb-gen/internal/merger"
	"awb-gen/internal/pipeline"
	"awb-gen/internal/profiler"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	debug     bool
	pprofAddr string
	input     string
	output    string
	useStdin  bool
	workers   int
	chunkSize int
)

var rootCmd = &cobra.Command{
	Use:   "awb-gen",
	Short: "High-Performance Go AWB Label Generator",
	Long: `awb-gen generates Air Waybill (AWB) PDFs from JSON input efficiently.
		
Examples:
  awb-gen --input data.json --output batch.pdf
  cat data.json | awb-gen --stdin --output batch.pdf`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		logger.InitLogger(debug)
		if pprofAddr != "" {
			profiler.Start(pprofAddr, logger.Log)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if logger.Log != nil {
			logger.Log.Fatal("awb-gen: fatal error", zap.Error(err))
		} else {
			fmt.Fprintln(os.Stderr, "awb-gen: fatal:", err)
		}
		os.Exit(1)
	}
	logger.Sync()
}

// run wires the input, starts the pipeline, and merges results.
func run() error {
	log := logger.Log
	start := time.Now()

	reader, cleanup, err := openInput()
	if err != nil {
		return err
	}
	defer cleanup()

	cfg := pipeline.Defaults()
	if workers > 0 {
		cfg.WorkerCount = workers
		// Respect MaxConcurrentPDF cap even if caller overrides WorkerCount.
		if workers < cfg.MaxConcurrentPDF {
			cfg.MaxConcurrentPDF = workers
		}
		cfg.JobBufferSize = workers * 2
		cfg.ResultBufferSize = workers * 4
	}
	if chunkSize > 0 {
		cfg.MergeChunkSize = chunkSize
	}

	log.Info("awb-gen: starting pipeline",
		zap.Int("workers", cfg.WorkerCount),
		zap.Int("max_concurrent_pdf", cfg.MaxConcurrentPDF),
		zap.Int("merge_chunk_size", cfg.MergeChunkSize),
		zap.String("output", output),
	)

	ctx := context.Background()

	pl := pipeline.New(cfg, log)
	results, err := pl.Run(ctx, reader)
	if err != nil {
		return fmt.Errorf("pipeline: %w", err)
	}

	// Merger writes directly to the output file.
	mg := merger.NewOrderedMerger(log, cfg.MergeChunkSize)
	count, err := mg.MergeToFile(results, output)
	if err != nil {
		return fmt.Errorf("merger: %w", err)
	}

	elapsed := time.Since(start)
	log.Info("awb-gen: done",
		zap.Int("labels", count),
		zap.String("output", output),
		zap.Duration("elapsed", elapsed),
		zap.Float64("labels_per_sec", float64(count)/elapsed.Seconds()),
	)
	return nil
}

// openInput returns a reader for the JSON input.
func openInput() (io.Reader, func(), error) {
	switch {
	case useStdin && input != "":
		return nil, func() {}, fmt.Errorf("--stdin and --input are mutually exclusive")

	case useStdin:
		logger.Log.Info("awb-gen: reading from stdin")
		return os.Stdin, func() {}, nil

	case input != "":
		f, err := os.Open(input)
		if err != nil {
			return nil, func() {}, fmt.Errorf("open %q: %w", input, err)
		}
		logger.Log.Info("awb-gen: reading from file", zap.String("path", input))
		return f, func() { _ = f.Close() }, nil

	default:
		return nil, func() {}, fmt.Errorf("provide --input <path> or --stdin")
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false,
		"Enable debug-level console logging")
	rootCmd.PersistentFlags().StringVar(&pprofAddr, "pprof", "",
		"Start pprof server at this address (e.g. localhost:6060)")

	rootCmd.Flags().StringVarP(&input, "input", "i", "",
		"Path to JSON input file containing an array of AWB objects")
	rootCmd.Flags().StringVarP(&output, "output", "o", "output.pdf",
		"Destination path for the generated PDF")
	rootCmd.Flags().BoolVar(&useStdin, "stdin", false,
		"Read JSON payload from standard input (pipe-friendly)")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", 0,
		"Number of parallel render workers (default: NumCPU)")
	rootCmd.Flags().IntVar(&chunkSize, "chunk-size", 0,
		"Pages per pdfcpu merge call — lower values reduce peak RSS (default: 500)")
}
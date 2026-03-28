package profiler

import (
	"net/http"
	_ "net/http/pprof" // side-effect: registers /debug/pprof/* on DefaultServeMux

	"go.uber.org/zap"
)

// Start launches a pprof HTTP server on addr in a background goroutine.
// It is a no-op when addr is empty. Callers pass logger.Log so that
// profiler startup and any server errors are captured in the structured log.
//
// Typical usage during development:
//
//	awb-gen --pprof localhost:6060 --input data.json --output out.pdf
//	go tool pprof http://localhost:6060/debug/pprof/heap
func Start(addr string, logger *zap.Logger) {
	if addr == "" {
		return
	}

	go func() {
		logger.Info("profiler: pprof server starting", zap.String("addr", addr))
		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Error("profiler: pprof server stopped", zap.Error(err))
		}
	}()
}
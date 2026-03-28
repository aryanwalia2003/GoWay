package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global logger instance. It is initialised by InitLogger before
// any pipeline work begins, then read-only for the rest of the process
// lifetime — safe to access from any goroutine without synchronisation.
var Log *zap.Logger

// InitLogger constructs and installs the global zap logger.
//
// In debug mode the output is human-readable coloured console text at DEBUG+.
// In production mode the output is structured JSON at INFO+, written to stdout
// so that log aggregators (Fluentd, CloudWatch, etc.) can consume it directly.
func InitLogger(debug bool) {
	enc := buildEncoder(debug)
	level := chooseLevel(debug)

	core := zapcore.NewCore(
		enc,
		zapcore.AddSync(os.Stdout),
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= level
		}),
	)

	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	zap.ReplaceGlobals(Log)
}

// Sync flushes any buffered log entries. Always defer this after InitLogger.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

func buildEncoder(debug bool) zapcore.Encoder {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	if debug {
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		return zapcore.NewConsoleEncoder(cfg)
	}
	return zapcore.NewJSONEncoder(cfg)
}

func chooseLevel(debug bool) zapcore.Level {
	if debug {
		return zapcore.DebugLevel
	}
	return zapcore.InfoLevel
}
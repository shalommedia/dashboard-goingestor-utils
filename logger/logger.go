package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	defaultLogger *slog.Logger
	once          sync.Once
)

// Config controls how the shared logger is initialized.
type Config struct {
	Level     string
	Service   string
	Output    io.Writer
	AddSource bool
	Format    string
}

// Default returns a lazily initialized shared structured logger.
func Default() *slog.Logger {
	once.Do(func() {
		defaultLogger = New(Config{})
	})

	return defaultLogger
}

// New creates a structured logger using sane defaults for Lambda-style services.
func New(cfg Config) *slog.Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	handlerOptions := &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     parseLevel(cfg.Level),
	}

	var handler slog.Handler
	if strings.EqualFold(strings.TrimSpace(cfg.Format), "text") {
		handler = slog.NewTextHandler(cfg.Output, handlerOptions)
	} else {
		handler = slog.NewJSONHandler(cfg.Output, handlerOptions)
	}

	logger := slog.New(handler)
	if cfg.Service != "" {
		logger = logger.With("service", cfg.Service)
	}

	return logger
}

// SetDefault replaces the shared logger instance and the process-wide slog default logger.
func SetDefault(logger *slog.Logger) {
	if logger == nil {
		return
	}

	defaultLogger = logger
	slog.SetDefault(logger)
}

// With returns a child logger with additional structured fields.
func With(args ...any) *slog.Logger {
	return Default().With(args...)
}

// WithContext returns the provided logger when present, otherwise the shared default logger.
func WithContext(_ context.Context, logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}

	return Default()
}

func parseLevel(level string) slog.Leveler {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

package observability

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger crea un logger JSON con nivel configurable via LOG_LEVEL (debug, info, warn, error).
func NewLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"

	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		levelStr = "info"
	}
	levelStr = strings.ToLower(levelStr)

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		return nil, fmt.Errorf("invalid LOG_LEVEL %q: %w", levelStr, err)
	}
	cfg.Level = zap.NewAtomicLevelAt(level)

	return cfg.Build()
}

// LoggerWithTrace enriquece el logger con trace_id/span_id si hay un span en el contexto.
func LoggerWithTrace(ctx context.Context, base *zap.Logger) *zap.Logger {
	if base == nil {
		return base
	}
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return base
	}

	return base.With(
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	)
}

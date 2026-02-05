package tracing

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

// InstrumentationName es el nombre base que usamos para crear tracers
// desde servicios que usan este módulo.
const InstrumentationName = "github.com/nuevo-idp/platform/tracing"

// Tracer devuelve un tracer compartido para el módulo de plataforma.
// Los servicios deberían usarlo para spans propios cuando no haya
// un nombre de paquete más específico.
func Tracer() trace.Tracer {
	return otel.Tracer(InstrumentationName)
}

// StartSpan es un helper liviano para crear spans hijos a partir de un ctx.
// No configura ningún exporter ni sampler: eso se resuelve en la inicialización
// global de OpenTelemetry del proceso (cuando la agreguemos).
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, opts...)
}

// InitTracing configura un TracerProvider global para el proceso.
// Si la variable de entorno OTEL_EXPORTER_OTLP_ENDPOINT está definida,
// se configura un exporter OTLP HTTP que enviará spans al collector.
// En caso contrario, se usa un provider sin exporter (no-op) para no
// fallar en desarrollo local.
//
// Devuelve una función de shutdown que debería llamarse al finalizar
// el proceso para flush de spans.
func InitTracing(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	if serviceName == "" {
		serviceName = "unknown-service"
	}

	var tp *sdktrace.TracerProvider

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		// Provider simple sin exporter; útil para dev cuando no hay collector.
		res, err := resource.New(ctx,
			resource.WithAttributes(
				semconv.ServiceName(serviceName),
			),
		)
		if err != nil {
			return func(context.Context) error { return nil }, err
		}
		p := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(p)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
		return p.Shutdown, nil
	}

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return func(context.Context) error { return nil }, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return func(context.Context) error { return nil }, err
	}

	tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithBatchTimeout(5*time.Second),
		),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp.Shutdown, nil
}

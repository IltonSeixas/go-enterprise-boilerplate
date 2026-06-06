package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Shutdown func(context.Context) error

func InitLogger(serviceName string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.InitialFields = map[string]any{"service": serviceName}
	return cfg.Build()
}

func InitTracing(ctx context.Context, serviceName, otlpEndpoint string) (Shutdown, error) {
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otlpEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	return func(ctx context.Context) error {
		shutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(shutCtx)
	}, nil
}

func InitPrometheus() (Shutdown, error) {
	exp, err := promexporter.New()
	if err != nil {
		return nil, err
	}

	mp := metric.NewMeterProvider(metric.WithReader(exp))
	otel.SetMeterProvider(mp)

	return func(ctx context.Context) error {
		shutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return mp.Shutdown(shutCtx)
	}, nil
}

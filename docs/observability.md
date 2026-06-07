# Observability

## Overview

Observability is built into the boilerplate from the start using **OpenTelemetry** — the vendor-neutral standard. You can export to any compatible backend (Jaeger, Grafana Tempo, Honeycomb, Datadog, AWS X-Ray) by changing a single environment variable.

The three pillars — **traces**, **metrics**, and **logs** — are correlated by trace ID so you can move seamlessly from a high-level metric spike to the exact trace, then to the log lines of the failing request.

---

## Traces

`internal/infrastructure/telemetry/setup.go` configures a global `TracerProvider` backed by an OTLP gRPC exporter, batching spans and tagging them with the service name via OpenTelemetry semantic conventions. Always-on sampling is used by default — tune `sdktrace.WithSampler` for high-traffic production deployments.

```go
func InitTracing(ctx context.Context, serviceName, otlpEndpoint string) (Shutdown, error) {
    exp, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(otlpEndpoint),
        otlptracegrpc.WithInsecure(),
    )
    // ...
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exp),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName(serviceName),
        )),
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
    )
    otel.SetTracerProvider(tp)
    // ...
}
```

The provider is registered globally via `otel.SetTracerProvider`, so any adapter or use case can obtain a tracer with `otel.Tracer(name)` and start spans propagated through `context.Context` — extend coverage by wrapping the operations you want visible in traces.

---

## Metrics

`internal/infrastructure/telemetry/setup.go` wires an OpenTelemetry `MeterProvider` to the Prometheus exporter bridge (`go.opentelemetry.io/otel/exporters/prometheus`), which feeds the default Prometheus registry. The HTTP router exposes that registry at `GET /metrics` via `promhttp.Handler()`.

```go
func InitPrometheus() (Shutdown, error) {
    exp, err := promexporter.New()
    // ...
    mp := metric.NewMeterProvider(metric.WithReader(exp))
    otel.SetMeterProvider(mp)
    // ...
}
```

```go
// interface/http/router.go
r.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

Out of the box this exposes Go runtime and process metrics from the default registry. Register custom counters, histograms and gauges via `otel.Meter(name)` (or directly against the Prometheus registry) to track business and request-level metrics as the application grows.

### Scrape config (Prometheus)

```yaml
scrape_configs:
  - job_name: go-api
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
```

---

## Logs

Structured JSON logs via `go.uber.org/zap`, configured in `internal/infrastructure/telemetry/setup.go` on top of `zap.NewProductionConfig()` with an ISO-8601 timestamp encoder and a `service` field injected into every entry.

```go
func InitLogger(serviceName string) (*zap.Logger, error) {
    cfg := zap.NewProductionConfig()
    cfg.EncoderConfig.TimeKey = "ts"
    cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    cfg.InitialFields = map[string]any{"service": serviceName}
    return cfg.Build()
}
```

### Log format (production)

```json
{
  "ts": "2024-01-15T10:30:00.123Z",
  "level": "info",
  "msg": "user registered",
  "service": "go-enterprise-boilerplate",
  "user_id": "01HN..."
}
```

### Log levels

| Level | Use |
|---|---|
| `error` | Unrecoverable failures — always paged |
| `warn` | Recoverable unexpected states |
| `info` | Business events (user registered, login succeeded) |
| `debug` | Development only — never in production |

`zap.NewProductionConfig()` defaults to `info` level and JSON encoding — appropriate for production out of the box.

---

## Configuration

| Variable | Default | Description |
|---|---|---|
| `OTLP_ENDPOINT` | `localhost:4317` | OTLP gRPC endpoint for traces and metrics |

---

## Local Development

Start a local Jaeger all-in-one instance to visualize traces:

```bash
docker compose up jaeger
```

Open `http://localhost:16686` to browse traces.

The `docker-compose.yml` in the repository root includes Jaeger, Prometheus, and Grafana pre-configured.

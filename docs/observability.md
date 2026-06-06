# Observability

## Overview

Observability is built into the boilerplate from the start using **OpenTelemetry** — the vendor-neutral standard. You can export to any compatible backend (Jaeger, Grafana Tempo, Honeycomb, Datadog, AWS X-Ray) by changing a single environment variable.

The three pillars — **traces**, **metrics**, and **logs** — are correlated by trace ID so you can move seamlessly from a high-level metric spike to the exact trace, then to the log lines of the failing request.

---

## Traces

Every HTTP request is automatically instrumented via `otelgin`. gRPC calls are instrumented via the `otelgrpc` interceptor. Use cases emit child spans via the context-propagated tracer.

### Setup

```go
// infrastructure/telemetry/setup.go
exporter, _ := otlptracegrpc.New(ctx,
    otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
    otlptracegrpc.WithInsecure(),
)

tp := trace.NewTracerProvider(
    trace.WithBatcher(exporter),
    trace.WithResource(resource.NewWithAttributes(
        semconv.SchemaURL,
        semconv.ServiceName(cfg.ServiceName),
    )),
)
otel.SetTracerProvider(tp)
```

### Manual spans in use cases

```go
func (uc *RegisterUser) Execute(ctx context.Context, input dto.RegisterInput) error {
    ctx, span := otel.Tracer("usecase").Start(ctx, "RegisterUser.Execute")
    defer span.End()

    // attach attributes (never sensitive data)
    span.SetAttributes(attribute.String("user.email_domain", emailDomain(input.Email)))

    // ...
}
```

---

## Metrics

Prometheus-format metrics are exposed at `GET /metrics` via `prometheus/client_golang`. The `otelgin` middleware records request count and latency histograms automatically.

### Available metrics

| Metric | Type | Description |
|---|---|---|
| `http_requests_total` | Counter | Total HTTP requests by method, path, status |
| `http_request_duration_seconds` | Histogram | Request latency by method and path |
| `http_requests_in_flight` | Gauge | Currently active requests |
| `db_query_duration_seconds` | Histogram | Database query latency by operation |

### Scrape config (Prometheus)

```yaml
scrape_configs:
  - job_name: go-api
    static_configs:
      - targets: ['localhost:3000']
    metrics_path: /metrics
```

---

## Logs

Structured JSON logs via `log/slog` (Go 1.21+ standard library). Every log line includes the trace ID and span ID, enabling correlation with distributed traces.

### Trace ID injection

```go
func logWithTrace(ctx context.Context, msg string, args ...any) {
    span := trace.SpanFromContext(ctx)
    sc := span.SpanContext()
    slog.InfoContext(ctx, msg,
        append(args,
            "trace_id", sc.TraceID().String(),
            "span_id",  sc.SpanID().String(),
        )...,
    )
}
```

### Log format (production)

```json
{
  "time": "2024-01-15T10:30:00.123Z",
  "level": "INFO",
  "msg": "user registered",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "user_id": "01HN..."
}
```

### Log levels

| Level | Use |
|---|---|
| `ERROR` | Unrecoverable failures — always paged |
| `WARN` | Recoverable unexpected states |
| `INFO` | Business events (user registered, login succeeded) |
| `DEBUG` | Development only — never in production |

Set via `LOG_LEVEL=info` environment variable.

---

## Configuration

| Variable | Default | Description |
|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4317` | OTLP gRPC endpoint |
| `OTEL_SERVICE_NAME` | `go-enterprise-boilerplate` | Service name in traces |
| `OTEL_SERVICE_VERSION` | — | Injected by CI from git tag |
| `LOG_LEVEL` | `info` | slog log level |

---

## Local Development

Start a local Jaeger all-in-one instance to visualize traces:

```bash
docker compose up jaeger
```

Open `http://localhost:16686` to browse traces.

The `docker-compose.yml` in the repository root includes Jaeger, Prometheus, and Grafana pre-configured.

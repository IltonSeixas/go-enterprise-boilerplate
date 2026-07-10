# Stage 1: build
FROM golang:1.26.5-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o boilerplate \
    ./cmd/server

# Stage 2: runtime
FROM scratch AS runtime

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/boilerplate /boilerplate

EXPOSE 8080 50051

# No HEALTHCHECK: the runtime image is built FROM scratch and has no shell
# or HTTP client to probe /health — rely on the orchestrator's liveness probe
# against GET /health instead.

USER 1001:1001

ENTRYPOINT ["/boilerplate"]

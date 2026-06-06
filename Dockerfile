# Stage 1: build
FROM golang:1.22-alpine AS builder

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

EXPOSE 3000 50051

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/boilerplate", "healthcheck"]

USER 1001:1001

ENTRYPOINT ["/boilerplate"]

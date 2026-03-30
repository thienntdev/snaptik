# ===== Build Stage =====
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/snaptiktok \
    ./cmd/server

# ===== Runtime Stage =====
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/snaptiktok .

# Copy templates and static files
COPY --from=builder /app/internal/templates ./internal/templates
COPY --from=builder /app/static ./static

# Create temp directory
RUN mkdir -p /app/tmp/downloads && chown -R appuser:appgroup /app

USER appuser

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:3000/api/health || exit 1

# Run
CMD ["./snaptiktok"]

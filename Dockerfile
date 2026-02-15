# Build stage
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /yatogm \
    ./cmd/yatogm/

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    supercronic \
    && addgroup -g 1000 -S yatogm \
    && adduser -u 1000 -S yatogm -G yatogm

# Create directories
RUN mkdir -p /etc/yatogm /data && chown -R yatogm:yatogm /etc/yatogm /data

# Copy binary
COPY --from=builder /yatogm /usr/local/bin/yatogm

# Copy default crontab
COPY crontab /etc/yatogm/crontab

# Data volume for state persistence
VOLUME /data

# Switch to non-root user
USER yatogm

# Health check: verify the binary is accessible
HEALTHCHECK --interval=5m --timeout=10s --start-period=30s --retries=3 \
    CMD /usr/local/bin/yatogm -version || exit 1

# Run with supercronic
ENTRYPOINT ["supercronic", "/etc/yatogm/crontab"]

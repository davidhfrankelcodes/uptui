# ── Build stage ────────────────────────────────────────────────────────────────
FROM golang:1.21-alpine AS builder

WORKDIR /src

# Cache module downloads before copying source
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /uptui ./cmd/uptui

# ── Runtime stage ──────────────────────────────────────────────────────────────
FROM alpine:3.19

# ca-certificates: needed for HTTPS checks
# tzdata: correct timestamps in history log
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /uptui /usr/local/bin/uptui

# Default paths — override via env vars in docker-compose / -e flags
ENV UPTUI_DATA_DIR=/data \
    UPTUI_CONFIG_DIR=/config \
    UPTUI_LISTEN_ADDR=0.0.0.0:29374

# Create default directories so the container starts cleanly even without mounts
RUN mkdir -p /data /config

EXPOSE 29374

ENTRYPOINT ["/usr/local/bin/uptui"]
CMD ["daemon"]

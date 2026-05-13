# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary (matches .goreleaser.yaml flags)
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /usr/local/bin/dolphin .

# ── Runtime ────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /usr/local/bin/dolphin /usr/local/bin/dolphin

# Default config directory
VOLUME /root/.dolphin

# SSH transport (optional)
EXPOSE 2222

# pprof / metrics (optional)
EXPOSE 6060 9090

ENTRYPOINT ["dolphin"]
CMD [""]

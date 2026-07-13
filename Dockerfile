FROM golang:1.26-bookworm AS builder

RUN apt-get update && apt-get install -y \
    wkhtmltopdf \
    gcc \
    pkg-config

WORKDIR /Bot

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build \
    -trimpath \
    -ldflags="-s -w -buildid= \
    -X main.Version=$(git describe --tags --dirty --always) \
    -X main.Commit=$(git rev-parse --short HEAD) \
    -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o sbot ./cmd/Bot


FROM debian:12-slim

RUN apt-get update && apt-get install -y \
    wkhtmltopdf \
    ca-certificates \
    fonts-dejavu \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /Bot

COPY --from=builder /Bot/sbot .
COPY --from=builder /Bot/config ./config

CMD ["./sbot"]


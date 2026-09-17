FROM golang:1.27-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    make \
    git \
    gcc \
    g++ \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

# Отдельный слой для зависимостей.
# Пересоберётся только при изменении go.mod/go.sum.
COPY go.mod go.sum ./

RUN go mod download

# Теперь исходники
COPY . .

RUN git config --global --add safe.directory /src

ARG BUILD_TIME

RUN mkdir -p /out && \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags "\
      -linkmode external \
      -extldflags '-static' \
      -s -w \
      -X main.Version=$(git describe --tags --dirty --always) \
      -X main.Commit=$(git rev-parse --short HEAD) \
      -X main.BuildTime=${BUILD_TIME}" \
    -o /out/sbot \
    ./cmd/bot/main.go
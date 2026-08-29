FROM golang:1.26-bookworm AS builder

RUN apt update && apt install -y \
    make \
    git \
    gcc \
    g++ \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY . .

RUN git config --global --add safe.directory /src

RUN CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -tags chromium \
    -ldflags "\
      -linkmode external \
      -extldflags '-static' \
      -s -w \
      -X main.Version=$(git describe --tags --dirty --always) \
      -X main.Commit=$(git rev-parse --short HEAD) \
	  -X main.BuildTime=$(BUILD_TIME)"\
    -o /out/sbot \
    ./cmd/bot/main.go

# Stage 1: Builder
FROM golang:1.26-bookworm AS builder

# Установка зависимостей сборки
RUN apt-get update && apt-get install -y --no-install-recommends \
    make \
    git \
    gcc \
    g++ \
    libc6-dev \
    musl-tools \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

# Кэшируем зависимости Go
COPY go.mod go.sum ./
RUN go mod download

# Копируем остальные исходники
COPY . .

# Настройка git для извлечения метаданных
RUN git config --global --add safe.directory /src

# Аргументы для внедрения метаданных сборки
ARG VERSION
ARG COMMIT
ARG BUILD_TIME

# Сборка статического бинарника с использованием musl-gcc для совместимости с Debian
RUN VERSION=${VERSION:-$(git describe --tags --dirty --always 2>/dev/null || echo "dev")} \
    && COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "none")} \
    && BUILD_TIME=${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)} \
    && CGO_ENABLED=1 \
           GOOS=linux \
           GOARCH=amd64 \
           go build \
           -ldflags "\
             -linkmode external \
             -extldflags '-static' \
             -s -w \
             -X main.Version=${VERSION} \
             -X main.Commit=${COMMIT} \
       	  -X main.BuildTime=${BUILD_TIME}"\
           -o sbot \
           ./cmd/bot/main.go


# Stage 2: Runtime
FROM debian:12-slim

# Установка необходимых рантайм-пакетов
RUN apt-get update && apt-get install -y --no-install-recommends \
    wkhtmltopdf \
    ca-certificates \
    fonts-dejavu \
    && rm -rf /var/lib/apt/lists/*

# Создание непривилегированного пользователя для безопасности
RUN groupadd -r bot && useradd -r -g bot -d /sbot bot

WORKDIR /sbot

# Копируем бинарник и конфигурацию (если она доступна в контексте)
COPY --from=builder /src/sbot .
COPY --from=builder /src/config ./config

# Устанавливаем права доступа
RUN chown -R bot:bot /sbot

# Запуск от имени пользователя bot
USER bot

EXPOSE 8080

ENTRYPOINT ["./sbot"]


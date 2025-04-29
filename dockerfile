FROM golang:1.23-alpine AS builder

WORKDIR /bot

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -buildid= -extldflags=-static" \
    -buildvcs=false \
    -o sbot ./cmd/bot


FROM gcr.io/distroless/static-debian12


WORKDIR /bot

COPY --from=builder /bot/sbot .
COPY --from=builder /bot/config/* ./config/


EXPOSE 8080

CMD ["./sbot"]


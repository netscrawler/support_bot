FROM golang:1.23-alpine AS builder

WORKDIR /bot

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux -ldflags="-s -w" go build -o sbot ./cmd/bot/main.go

FROM gcr.io/distroless/static-debian12


WORKDIR /bot

COPY --from=builder /bot/sbot .
COPY --from=builder /bot/config/* ./config/


EXPOSE 54821

CMD ["./sbot"]


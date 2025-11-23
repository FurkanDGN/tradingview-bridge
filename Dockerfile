FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o tv-binance-bot main.go

FROM alpine:3.20

WORKDIR /root/

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/tv-binance-bot /usr/local/bin/tv-binance-bot

EXPOSE 8080

ENV PORT=8080

CMD ["tv-binance-bot"]

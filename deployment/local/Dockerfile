FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o hezzl-goods ./cmd/hezzl-goods/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/hezzl-goods .

EXPOSE 8081
CMD ["./hezzl-goods"]
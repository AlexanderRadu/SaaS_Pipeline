FROM golang:1.26-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /status-service ./cmd/status/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /status-service .

EXPOSE 8083

CMD ["./status-service"]
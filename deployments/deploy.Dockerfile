FROM golang:1.26-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /deploy-service ./cmd/deploy/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /deploy-service .

EXPOSE 8082

CMD ["./deploy-service"]
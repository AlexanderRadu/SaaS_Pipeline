FROM golang:1.26-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /catalog-service ./cmd/catalog/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /catalog-service .

EXPOSE 4040

CMD ["./catalog-service"]
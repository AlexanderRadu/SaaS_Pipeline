FROM golang:1.26-alpine as builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /migrate ./cmd/migrate/main.go

FROM alpine:latest

COPY --from=builder /migrate /migrate

ENTRYPOINT ["/migrate"]
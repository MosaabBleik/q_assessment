# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0

# ✅ Build from correct entry point
RUN go build -o inventory-service ./cmd/server

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/inventory-service .

EXPOSE 8081

CMD ["./inventory-service"]

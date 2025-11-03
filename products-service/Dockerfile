# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy all source files
COPY . .

# Disable CGO for small static binary
ENV CGO_ENABLED=0

# ✅ Build from the proper path
RUN go build -o products-service ./cmd/server

# --- Runtime stage ---
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/products-service .

# Optional environment variables
ENV PORT=8080
ENV REDIS_URL=redis://redis:6379

EXPOSE 8080

CMD ["./products-service"]

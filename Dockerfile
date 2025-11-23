# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o pr-api ./cmd/server/main.go

# Stage 2: Run
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/pr-api .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./pr-api"]
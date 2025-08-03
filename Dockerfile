# === Stage 1: Build binary ===
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .

RUN apk add --no-cache gcc musl-dev
RUN go mod tidy
RUN go build -o bot

# === Stage 2: Minimal runtime ===
FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/bot .

# folosim volum pentru DB (persistență)
VOLUME ["/root/data"]

CMD ["./bot"]

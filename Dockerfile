# Stage 1: Build binary
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

# instalăm pachete pentru compilare
RUN apk add --no-cache gcc musl-dev

# pregătim și construim binarul
RUN go mod tidy
RUN go build -o bot

# Stage 2: Minimal runtime
FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/bot .

# volum pentru baza de date (persistență)
VOLUME ["/root/data"]

CMD ["./bot"]

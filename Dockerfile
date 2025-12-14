# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o quran-bot ./cmd/bot

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata ffmpeg

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/quran-bot .

# Copy locales
COPY --from=builder /app/locales ./locales

# Create config directory
RUN mkdir -p /app/config

# Expose health check port (if needed)
EXPOSE 8080

# Run the bot
CMD ["./quran-bot"]

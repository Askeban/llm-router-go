# Multi-stage build for Go service
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o router cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/router .
COPY --from=builder /app/internal/models.json ./internal/models.json

# Create directories for data
RUN mkdir -p /data /configs

# Environment variables
ENV DATABASE_PATH=/data/router.db
ENV MODEL_PROFILES_PATH=./internal/models.json
ENV PORT=8080
ENV GIN_MODE=release

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./router"]
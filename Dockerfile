# Multi-stage build for Go service
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Copy source code
COPY . .

# Generate go.sum and download dependencies
RUN go mod tidy && go mod download

# Build the application
RUN go build -o router main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/router .

# Copy required files
COPY --from=builder /app/configs/model_1.json ./configs/model_1.json
COPY --from=builder /app/database/schema_postgres.sql ./database/schema_postgres.sql

# Create directories for data
RUN mkdir -p /data /configs /database

# Environment variables (Cloud SQL via Unix socket)
ENV MODEL_PATH=./configs/model_1.json
ENV SCHEMA_PATH=./database/schema_postgres.sql
ENV PORT=8080
ENV GIN_MODE=release

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./router"]
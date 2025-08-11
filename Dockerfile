## Build stage
FROM golang:1.20-alpine AS builder

WORKDIR /src
COPY . .
# Build the router binary
RUN go build -o llm-router ./cmd/router

## Final runtime stage
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /src/llm-router /usr/local/bin/llm-router
COPY config/models.yaml /config/models.yaml
ENV ROUTER_CONFIG=/config/models.yaml
EXPOSE 8080
ENTRYPOINT ["llm-router"]
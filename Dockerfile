# Build stage
FROM golang:1.25.3-bookworm AS builder

WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary
RUN go build -v -trimpath -o awb-gen main.go

# Run stage
FROM debian:bookworm-slim

WORKDIR /app

# Install CA certificates for any HTTPS requests
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/awb-gen /app/awb-gen

# Copy templates and assets
COPY --from=builder /app/templates /app/templates

# Default port
EXPOSE 8080

# Run the server
ENTRYPOINT ["/app/awb-gen", "serve", "--port", "8080"]

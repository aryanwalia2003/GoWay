# Build stage
FROM golang:1.25.3-bookworm AS builder

WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary
RUN go build -v -trimpath -ldflags="-s -w" -o awb-gen main.go

# Run stage using Alpine for minimal footprint.
FROM alpine:latest

# Install CA certificates and timezone data.
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/awb-gen /app/awb-gen

# Copy templates and assets
COPY --from=builder /app/templates /app/templates

# Default port
EXPOSE 8080

# Run the server
ENTRYPOINT ["/app/awb-gen", "serve", "--port", "8080"]

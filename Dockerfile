# Stage 1: Build
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 ensures a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o paas-server ./cmd/server

# Stage 2: Final
FROM alpine:latest

# Install CA certificates for HTTPS and tzdata for timezones
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/paas-server .

# Copy necessary assets (HTML templates are still needed)
COPY --from=builder /app/internal/delivery/html ./internal/delivery/html

# Expose the default port (matching config.yml)
EXPOSE 8080

# Run the server
CMD ["./paas-server"]

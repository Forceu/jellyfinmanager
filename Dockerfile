# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with docker tag
RUN CGO_ENABLED=0 GOOS=linux go build -tags docker -a -installsuffix cgo \
    -ldflags '-extldflags "-static" -s -w' \
    -o jellyfinmanager .

# Final stage
FROM scratch

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary from builder
COPY --from=builder /build/jellyfinmanager /jellyfinmanager

# Create backup directory
VOLUME ["/backup"]

# Set working directory
WORKDIR /

# Run as non-root user
USER 65534:65534

# Set the entrypoint
ENTRYPOINT ["/jellyfinmanager"]

# Default command (show help)
CMD ["-h"]

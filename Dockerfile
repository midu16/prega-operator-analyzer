# Multi-stage build for prega-operator-analyzer
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o prega-operator-analyzer ./cmd

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates git tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/prega-operator-analyzer .

# Create directories for volume mounts
RUN mkdir -p /app/output && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port (if needed for health checks)
EXPOSE 8080

# Set environment variables
ENV OUTPUT_DIR=/app/output
ENV PREGA_INDEX=quay.io/prega/test/prega/prega-operator-index:v4.20-20250908T090030

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["./prega-operator-analyzer", "--help"] || exit 1

# Default command - only output to mounted volume
ENTRYPOINT ["./prega-operator-analyzer"]

# Default arguments - use the specified Prega index and output to volume
CMD ["--prega-index=quay.io/prega/test/prega/prega-operator-index:v4.20-20250908T090030", "--output=/app/output/release-notes.txt", "--verbose"]

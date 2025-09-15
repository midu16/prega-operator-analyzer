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

# Final stage - Use OPM base image
FROM quay.io/operator-framework/opm:latest

# Install additional dependencies
RUN microdnf install -y git ca-certificates tzdata bash curl tar && \
    microdnf clean all

# Create non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/bash appuser

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
ENV PREGA_INDEX=quay.io/prega/prega-operator-index:v4.20-20250909T144138
ENV OUTPUT_FILE=release-notes.txt
ENV VERBOSE=true

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["./prega-operator-analyzer", "--help"] || exit 1

# Create a startup script that properly expands environment variables
RUN echo '#!/bin/bash' > /app/start.sh && \
    echo 'exec ./prega-operator-analyzer --prega-index="${PREGA_INDEX}" --output="/app/output/${OUTPUT_FILE}" --verbose' >> /app/start.sh && \
    chmod +x /app/start.sh

# Use the startup script as entrypoint
ENTRYPOINT ["/app/start.sh"]

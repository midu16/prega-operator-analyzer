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

# Final stage - Use UBI9 standard (not minimal) which includes shell and basic tools
FROM registry.access.redhat.com/ubi9/ubi:latest

# Install OPM from the official image
COPY --from=quay.io/operator-framework/opm:v1.48.0 /bin/opm /usr/local/bin/opm

# Install additional dependencies using dnf (standard UBI uses dnf, not microdnf)
RUN dnf install -y git ca-certificates tzdata bash curl tar shadow-utils && \
    dnf clean all

# Create non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/bash appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/prega-operator-analyzer .

# Create directories for volume mounts
RUN mkdir -p /app/output && \
    chown -R appuser:appgroup /app && \
    chmod 777 /app/output

# Switch to non-root user
USER appuser

# Expose port (if needed for health checks)
EXPOSE 8080

# Set environment variables
ENV OUTPUT_DIR=/app/output
ENV PREGA_INDEX=quay.io/prega/prega-operator-index:v4.21-20251025T205504
ENV OUTPUT_FILE=release-notes.txt
ENV VERBOSE=true

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["./prega-operator-analyzer", "--help"] || exit 1

# Create a startup script that properly expands environment variables and ensures output directory is writable
RUN echo '#!/bin/bash' > /app/start.sh && \
    echo '' >> /app/start.sh && \
    echo '# Ensure output directory exists and is writable' >> /app/start.sh && \
    echo 'mkdir -p /app/output 2>/dev/null || true' >> /app/start.sh && \
    echo 'touch /app/output/.test 2>/dev/null && rm -f /app/output/.test 2>/dev/null || {' >> /app/start.sh && \
    echo '  echo "ERROR: /app/output directory is not writable. Please ensure volume is mounted with appropriate permissions."' >> /app/start.sh && \
    echo '  echo "Try using :z or :Z suffix when mounting volume (e.g., -v \$(pwd)/output:/app/output:Z)"' >> /app/start.sh && \
    echo '  exit 1' >> /app/start.sh && \
    echo '}' >> /app/start.sh && \
    echo '' >> /app/start.sh && \
    echo '# If arguments are provided, pass them to the application' >> /app/start.sh && \
    echo '# Otherwise, use default environment variable configuration' >> /app/start.sh && \
    echo 'if [ $# -gt 0 ]; then' >> /app/start.sh && \
    echo '  exec ./prega-operator-analyzer "$@"' >> /app/start.sh && \
    echo 'else' >> /app/start.sh && \
    echo '  exec ./prega-operator-analyzer --prega-index="${PREGA_INDEX}" --output="/app/output/${OUTPUT_FILE}" --verbose' >> /app/start.sh && \
    echo 'fi' >> /app/start.sh && \
    chmod +x /app/start.sh

# Use the startup script as entrypoint
ENTRYPOINT ["/app/start.sh"]

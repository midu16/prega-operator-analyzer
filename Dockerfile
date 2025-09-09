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
ENV PREGA_INDEX=quay.io/prega/prega-operator-index:v4.20-20250909T144138
ENV OUTPUT_FILE=release-notes.txt
ENV VERBOSE=true

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["./prega-operator-analyzer", "--help"] || exit 1

# Default command - use environment variables for configuration
ENTRYPOINT ["./prega-operator-analyzer"]

# Default arguments - use environment variables
CMD ["--prega-index=${PREGA_INDEX}", "--output=/app/output/${OUTPUT_FILE}", "--verbose"]

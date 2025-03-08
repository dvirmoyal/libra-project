# Stage 1: Build {check new security update of the image}
FROM golang:1.22.1-alpine3.19@sha256:43c094ad24b6ac0546c62193baeb3e6e49ce14d3250845d166c77c25f64b0386 AS builder

# Set working directory
WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build with security flags
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o server ./cmd/

# Stage 2: Create minimal runtime image
FROM alpine:3.19.1@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b


# Create non-root user only
RUN addgroup -g 1001 appgroup && \
    adduser -u 1001 -g appgroup -S appuser

# Create config directory with proper permissions
RUN mkdir /config && chown -R appuser:appgroup /config

# Copy config from builder stage
COPY --from=builder --chown=appuser:appgroup /app/config /config

# Copy the binary from builder stage
COPY --from=builder --chown=appuser:appgroup /app/server /server

# Use non-root user
USER appuser:appgroup

# Expose the API port
EXPOSE 8080

# Run the service
ENTRYPOINT ["/server"]
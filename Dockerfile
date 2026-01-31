# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files only (for better caching)
COPY go.mod go.sum ./

# Download dependencies - this layer is cached as long as go.mod/go.sum don't change
RUN go mod download

# Copy source code
COPY cmd ./cmd
COPY internal ./internal

# Build the application with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o auto-rename ./cmd/auto-rename

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/auto-rename .

# Copy static assets
COPY template ./template
COPY static ./static

# Create required directories
RUN mkdir -p /app/files /app/data

# Make the binary executable
RUN chmod +x ./auto-rename

# Expose port for web interface
EXPOSE 8080

# Set the default command
ENTRYPOINT ["./auto-rename"]

# Default arguments - start web server only
# CMD ["-web-only", "-db=/app/data/renames.db", "-web-port=8080"]

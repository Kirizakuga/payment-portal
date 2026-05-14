# Build stage
FROM golang:1.21.5-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Copy templates and static files
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Copy credentials if they exist (will fail if missing and not handled)
# Using a wildcard to make it optional during build if excluded by .dockerignore
COPY --from=builder /app/google-credentials.json* ./

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./main"]

# 🏗️ Stage 1: Build the Go app using full Go image
FROM golang:1.24 as builder

WORKDIR /app

# Copy go.mod and go.sum first (to leverage Docker layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# 🐳 Stage 2: Create minimal runtime image
FROM alpine:latest

# Install certificates so HTTPS works inside containers
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /app/server .

# Expose the application's port
EXPOSE 8080

# Command to run the app
CMD ["./server"]

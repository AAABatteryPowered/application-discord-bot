# Use official Go image to build the app
FROM golang:1.23-alpine AS builder

# Install git (needed for go modules sometimes)
RUN apk add --no-cache git

# Set working directory inside container
WORKDIR /app

# Copy go.mod and go.sum files first (for caching dependencies)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of your source code
COPY . .

# Build the bot binary
RUN go build -o bot .

# ---- Final image ----
FROM alpine:latest

# Copy CA certificates (needed for HTTPS)
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy the compiled binary from builder
COPY --from=builder /app/bot .

# Command to run your bot
CMD ["./bot"]

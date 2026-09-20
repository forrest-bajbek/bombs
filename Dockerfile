# Stage 1: Build the binary
FROM golang:1.27.1-alpine3.24 AS builder

# Set working directory
WORKDIR /app

# Copy dependency files first to leverage Docker caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary (statically linked for smaller/portable images)
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Stage 2: Create the final lean image
FROM alpine:latest

WORKDIR /bombs/

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/main .

# Expose the application port (e.g., 8080)
EXPOSE 8080

# Run the binary
CMD ["./main"]

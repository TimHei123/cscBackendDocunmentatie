# Use official Golang image to build the application
FROM golang:1.23-alpine as builder

# Set the working directory inside the container
WORKDIR /app

# Copy go modules files
COPY go.mod go.sum ./

# Download the Go modules
RUN go mod tidy

# Copy the entire Go project into the container
COPY . .

# Build the Go application
RUN go build -o main .

# Start a new stage for the final image
FROM alpine:latest

# Install required packages (for running the Go application)
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the built binary from the previous stage
COPY --from=builder /app/main /app/main

# Expose port 8080 for the Echo server
EXPOSE 8080

# Command to run the Go application
CMD ["./main"]

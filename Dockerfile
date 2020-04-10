# Start from the latest golang base image
FROM golang:1.14.1

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY src ./

# Build the Go app
RUN go build -o main .

# Expose port 8080 to the outside world
EXPOSE 4443

# Command to run the executable
CMD ["./main"]
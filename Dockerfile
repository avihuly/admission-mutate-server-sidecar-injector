# Start from the latest golang base image
FROM golang:1.14.1

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY src ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Build the Go app
RUN go build -o main .

# Expose port 8080 to the outside world
EXPOSE 4443

# Command to run the executable
CMD ["./main"]
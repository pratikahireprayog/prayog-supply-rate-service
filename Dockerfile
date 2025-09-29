FROM golang:1.25.1-alpine

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o supply-rate ./cmd/server

# Expose the application port
EXPOSE 9046

# Command to run the executable
CMD ["./supply-rate"] 
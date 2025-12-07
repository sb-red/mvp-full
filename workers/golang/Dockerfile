FROM golang:1.21-alpine

WORKDIR /app

# Copy go.mod first
COPY go.mod ./

# Copy source files
COPY *.go ./

# Download dependencies and generate go.sum
RUN go mod tidy

# Build the worker binary
RUN go build -o worker .

# Create sandbox directory
RUN mkdir -p /tmp/sandbox /tmp/gocache /tmp/gopath

CMD ["./worker"]

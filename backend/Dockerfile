# Build stage
FROM golang:1.21-alpine AS build

WORKDIR /app

# Install swag for swagger docs generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy go mod file
COPY go.mod ./

# Download dependencies (generates go.sum automatically)
RUN go mod download

# Copy source code
COPY . .

# Generate swagger docs and build the application
RUN swag init && go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o server .

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Copy binary from build stage
COPY --from=build /app/server .

# Expose port
EXPOSE 8080

# Run
ENTRYPOINT ["./server"]

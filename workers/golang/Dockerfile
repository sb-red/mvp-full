# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

COPY go.mod ./
COPY *.go ./

RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o worker .

# Runtime stage
FROM golang:1.21-alpine

# Create non-root user
RUN addgroup -g 1000 -S worker && \
    adduser -u 1000 -S worker -G worker -h /app

WORKDIR /app

# Copy built worker binary
COPY --from=builder --chown=worker:worker /build/worker .

# Switch to non-root user
USER worker

ENV GOCACHE=/tmp/gocache
ENV GOPATH=/tmp/gopath

CMD ["./worker"]

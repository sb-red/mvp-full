# Build stage
FROM swift:5.9-jammy AS builder

WORKDIR /build

# Copy package files
COPY Package.swift .
COPY Sources ./Sources

# Build the application
RUN swift build -c release

# Runtime stage
FROM swift:5.9-jammy

# Create non-root user
RUN groupadd -r -g 1000 worker && \
    useradd -r -u 1000 -g worker -d /app -s /sbin/nologin worker

WORKDIR /app

# Copy built binary
COPY --from=builder --chown=worker:worker /build/.build/release/swift-worker .

# Switch to non-root user
USER worker

CMD ["./swift-worker"]

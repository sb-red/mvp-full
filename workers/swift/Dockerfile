FROM swift:5.9-jammy

WORKDIR /app

# Copy package files
COPY Package.swift .
COPY Sources ./Sources

# Build the application
RUN swift build -c release

CMD [".build/release/swift-worker"]

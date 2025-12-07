# Build stage
FROM rust:1.73 AS builder

ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:$PATH

RUN rustup default stable

WORKDIR /build

# Copy Cargo files first for better caching
COPY Cargo.toml ./

# Create src directory and a dummy main.rs for dependency caching
RUN mkdir -p src && echo "fn main() {}" > src/main.rs

# Build dependencies only
RUN cargo build --release || true

# Now copy actual source code and rebuild
COPY src/ src/

# Touch source files to force rebuild
RUN touch src/*.rs

# Build the worker
RUN cargo build --release

# Runtime stage
FROM rust:1.73-slim

# Install minimal runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user (use 10001 to avoid conflicts)
RUN groupadd -r -g 10001 worker && \
    useradd -r -u 10001 -g worker -d /app -s /sbin/nologin worker

WORKDIR /app

# Copy built binary
COPY --from=builder --chown=worker:worker /build/target/release/rust-worker .

# Switch to non-root user
USER worker

CMD ["./rust-worker"]

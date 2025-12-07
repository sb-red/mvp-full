FROM rust:1.73

# Set up Rust environment
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:$PATH

# Ensure stable toolchain is default
RUN rustup default stable

WORKDIR /app

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

# Create sandbox directory
RUN mkdir -p /tmp/sandbox

CMD ["./target/release/rust-worker"]

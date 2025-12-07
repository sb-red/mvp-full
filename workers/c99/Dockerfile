FROM gcc:12

# Install dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends python3 python3-pip python3-venv git cmake && \
    rm -rf /var/lib/apt/lists/*

# Create virtual environment and install redis
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
RUN pip install --no-cache-dir redis

# Install cJSON library to /usr prefix
RUN git clone --depth 1 https://github.com/DaveGamble/cJSON.git /tmp/cJSON && \
    cd /tmp/cJSON && \
    mkdir build && cd build && \
    cmake -DCMAKE_INSTALL_PREFIX=/usr .. && make && make install && \
    ldconfig && \
    rm -rf /tmp/cJSON

# Create non-root user (use 10001 to avoid conflicts)
RUN groupadd -r -g 10001 worker && \
    useradd -r -u 10001 -g worker -d /app -s /sbin/nologin worker

WORKDIR /app

# Copy source files with correct ownership
COPY --chown=worker:worker . .

# Switch to non-root user
USER worker

CMD ["python", "worker.py"]

FROM silkeh/clang:15

# Install dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends python3 python3-pip python3-venv curl && \
    rm -rf /var/lib/apt/lists/*

# Create virtual environment and install redis
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
RUN pip install --no-cache-dir redis

# Install nlohmann/json header-only library
RUN mkdir -p /usr/include/nlohmann /usr/local/include/nlohmann && \
    curl -L https://github.com/nlohmann/json/releases/download/v3.11.2/json.hpp \
    -o /usr/include/nlohmann/json.hpp && \
    cp /usr/include/nlohmann/json.hpp /usr/local/include/nlohmann/json.hpp

# Create non-root user
RUN groupadd -r -g 1000 worker && \
    useradd -r -u 1000 -g worker -d /app -s /sbin/nologin worker

WORKDIR /app

# Copy source files with correct ownership
COPY --chown=worker:worker . .

# Switch to non-root user
USER worker

CMD ["python", "worker.py"]

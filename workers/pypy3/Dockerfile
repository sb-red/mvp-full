FROM pypy:3.10-slim

# Install dependencies as root
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Create non-root user (use 10001 to avoid conflicts)
RUN groupadd -r -g 10001 worker && \
    useradd -r -u 10001 -g worker -d /app -s /sbin/nologin worker

# Copy source files with correct ownership
COPY --chown=worker:worker . .

# Switch to non-root user
USER worker

CMD ["pypy3", "worker.py"]

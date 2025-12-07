FROM python:3.9-slim

# Install dependencies as root
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Create non-root user
RUN groupadd -r -g 1000 worker && \
    useradd -r -u 1000 -g worker -d /app -s /sbin/nologin worker

# Copy source files with correct ownership
COPY --chown=worker:worker . .

# Switch to non-root user
USER worker

CMD ["python", "worker.py"]

FROM ruby:3.2-slim

# Install build dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends build-essential && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install gems as root
COPY Gemfile ./
RUN bundle install

# Create non-root user
RUN groupadd -r -g 1000 worker && \
    useradd -r -u 1000 -g worker -d /app -s /sbin/nologin worker

# Copy source files with correct ownership
COPY --chown=worker:worker . .

# Switch to non-root user
USER worker

CMD ["ruby", "worker.rb"]

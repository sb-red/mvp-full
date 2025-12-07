FROM node:20-alpine

# Install dependencies as root
WORKDIR /app
COPY package.json ./
RUN npm install --production && npm cache clean --force

# Create non-root user (Alpine uses addgroup/adduser, use 10001 to avoid conflicts)
RUN addgroup -g 10001 -S worker && \
    adduser -u 10001 -S worker -G worker -h /app

# Copy source files with correct ownership
COPY --chown=worker:worker . .

# Switch to non-root user
USER worker

CMD ["node", "worker.js"]

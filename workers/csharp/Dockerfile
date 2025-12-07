# Build stage
FROM mcr.microsoft.com/dotnet/sdk:7.0 AS builder

WORKDIR /build

# Copy csproj first for better caching
COPY Worker.csproj ./

# Restore dependencies
RUN dotnet restore

# Copy source files
COPY *.cs ./

# Build the worker
RUN dotnet publish -c Release -o out

# Runtime stage
FROM mcr.microsoft.com/dotnet/runtime:7.0

# Create non-root user (use 10001 to avoid conflicts)
RUN groupadd -r -g 10001 worker && \
    useradd -r -u 10001 -g worker -d /app -s /sbin/nologin worker

WORKDIR /app

# Copy published output
COPY --from=builder --chown=worker:worker /build/out .

# Switch to non-root user
USER worker

CMD ["dotnet", "Worker.dll"]

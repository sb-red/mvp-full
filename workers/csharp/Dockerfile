FROM mcr.microsoft.com/dotnet/sdk:7.0

WORKDIR /app

# Copy csproj first for better caching
COPY Worker.csproj ./

# Restore dependencies
RUN dotnet restore

# Copy source files
COPY *.cs ./

# Build the worker
RUN dotnet publish -c Release -o out

# Create sandbox directory
RUN mkdir -p /tmp/sandbox

CMD ["dotnet", "out/Worker.dll"]

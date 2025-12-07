FROM eclipse-temurin:11-jdk-jammy

# Install Maven
RUN apt-get update && \
    apt-get install -y --no-install-recommends maven && \
    rm -rf /var/lib/apt/lists/*

# Pre-download Gson JAR for network-isolated environment
RUN mkdir -p /opt/libs && \
    wget -q -O /opt/libs/gson-2.10.1.jar \
    https://repo1.maven.org/maven2/com/google/code/gson/gson/2.10.1/gson-2.10.1.jar

ENV GSON_JAR_PATH=/opt/libs/gson-2.10.1.jar

WORKDIR /app

# Copy project files
COPY pom.xml .
COPY src ./src

# Build the application
RUN mvn clean package -DskipTests

# Create non-root user
RUN groupadd -r -g 1000 worker && \
    useradd -r -u 1000 -g worker -d /app -s /sbin/nologin worker

# Set ownership
RUN chown -R worker:worker /app

# Switch to non-root user
USER worker

CMD ["java", "-jar", "target/java11-worker-1.0.0.jar"]

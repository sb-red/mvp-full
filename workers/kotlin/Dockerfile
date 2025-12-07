FROM eclipse-temurin:17-jdk-jammy

# Install Kotlin compiler, Gradle, and pre-download Gson
RUN apt-get update && apt-get install -y --no-install-recommends wget unzip curl && \
    # Install Kotlin compiler
    wget -q https://github.com/JetBrains/kotlin/releases/download/v1.9.20/kotlin-compiler-1.9.20.zip && \
    unzip kotlin-compiler-1.9.20.zip -d /opt && \
    rm kotlin-compiler-1.9.20.zip && \
    # Install Gradle
    wget -q https://services.gradle.org/distributions/gradle-8.4-bin.zip && \
    unzip gradle-8.4-bin.zip -d /opt && \
    rm gradle-8.4-bin.zip && \
    # Pre-download Gson JAR for network-isolated environment
    mkdir -p /opt/libs && \
    wget -q -O /opt/libs/gson-2.10.1.jar https://repo1.maven.org/maven2/com/google/code/gson/gson/2.10.1/gson-2.10.1.jar && \
    rm -rf /var/lib/apt/lists/*

ENV PATH="/opt/kotlinc/bin:/opt/gradle-8.4/bin:${PATH}"
ENV GSON_JAR_PATH="/opt/libs/gson-2.10.1.jar"

WORKDIR /app

# Copy project files
COPY build.gradle.kts .
COPY src ./src

# Build the application
RUN gradle jar --no-daemon

# Create non-root user (use 10001 to avoid conflicts)
RUN groupadd -r -g 10001 worker && \
    useradd -r -u 10001 -g worker -d /app -s /sbin/nologin worker

# Set ownership
RUN chown -R worker:worker /app

# Switch to non-root user
USER worker

CMD ["java", "-jar", "build/libs/kotlin-1.0.0.jar"]

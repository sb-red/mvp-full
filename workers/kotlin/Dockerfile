FROM eclipse-temurin:17-jdk-jammy

# Install Kotlin compiler and Gradle
RUN apt-get update && apt-get install -y wget unzip curl && \
    # Install Kotlin compiler
    wget -q https://github.com/JetBrains/kotlin/releases/download/v1.9.20/kotlin-compiler-1.9.20.zip && \
    unzip kotlin-compiler-1.9.20.zip -d /opt && \
    rm kotlin-compiler-1.9.20.zip && \
    # Install Gradle
    wget -q https://services.gradle.org/distributions/gradle-8.4-bin.zip && \
    unzip gradle-8.4-bin.zip -d /opt && \
    rm gradle-8.4-bin.zip && \
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

CMD ["java", "-jar", "build/libs/kotlin-1.0.0.jar"]

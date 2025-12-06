FROM eclipse-temurin:21-jdk-jammy

# Install Maven and wget
RUN apt-get update && apt-get install -y maven wget && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy project files
COPY pom.xml .
COPY src ./src

# Build the application
RUN mvn clean package -DskipTests

CMD ["java", "-jar", "target/java21-worker-1.0.0.jar"]

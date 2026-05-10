FROM gradle:8.7-jdk21 AS builder
WORKDIR /app
COPY build.gradle settings.gradle* gradle.properties* ./
COPY gradle ./gradle
COPY gradlew ./
RUN ./gradlew dependencies --no-daemon -q || true
COPY src ./src
RUN ./gradlew bootJar --no-daemon -x test

FROM eclipse-temurin:21-jre-jammy
WORKDIR /app
# Firefox для Playwright (Java)
RUN apt-get update && apt-get install -y firefox && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/build/libs/*.jar app.jar
EXPOSE 8080
ENTRYPOINT ["java", "-XX:+UseContainerSupport", "-XX:MaxRAMPercentage=75.0", "-jar", "app.jar"]

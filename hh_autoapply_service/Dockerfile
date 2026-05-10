FROM golang:1.22-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o autoapply ./cmd/main.go

FROM debian:bookworm-slim
WORKDIR /app
# Chromium для playwright-go
RUN apt-get update && apt-get install -y \
    chromium ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/autoapply .
ENV PLAYWRIGHT_BROWSERS_PATH=/usr/bin
EXPOSE 8081
CMD ["./autoapply"]

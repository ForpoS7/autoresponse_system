FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o auth_service ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/auth_service .
COPY config.yml .
EXPOSE 8082
CMD ["./auth_service"]

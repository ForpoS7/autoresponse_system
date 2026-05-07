# C4 Level 2 — Container Diagram

Все контейнеры (процессы, хранилища) внутри системы и их связи.

```mermaid
C4Container
  title Container Diagram — HH AutoResponse System

  Person(user, "Соискатель")

  System_Boundary(hh_auto, "HH AutoResponse System") {

    Container(nginx, "nginx", "nginx:alpine", "API Gateway: маршрутизация, rate limiting (5 rpm auth / 60 rpm api), load balancing least_conn")

    Container(auth_svc, "auth_service", "Go 1.22 · Docker :8082", "Регистрация, вход, валидация JWT. BCrypt пароли, HS256 токены")
    Container(aggregate_svc, "hh_aggregate_service", "Java 21 · Spring Boot 3.2.5 · Host :8080", "Парсинг вакансий через Playwright (Firefox). Публикует результаты в Kafka")
    Container(autoapply_svc, "hh_autoapply_service", "Go 1.22 · Host :8081", "EDA-оркестрация автоотклика. Playwright для отправки. Архивация в MinIO")
    Container(generate_svc, "generate_service", "Python · Host", "Kafka consumer: vacancy_input → Ollama → vacancy_output")

    ContainerDb(postgres, "PostgreSQL", "postgres:16 · :5444", "users, hh_tokens, auto_apply_requests, auto_apply_logs, vacancies")
    ContainerDb(kafka, "Apache Kafka", "KRaft 3.7 · :9092", "5 топиков: parse.requested, vacancies.parsed, token.updated, vacancy_input, vacancy_output")
    ContainerDb(redis, "Redis", "redis:7 · :6379", "Кеш HH-токенов (24h TTL) и статусов автоотклика (1h / terminal ∞)")
    ContainerDb(minio, "MinIO", "S3-compatible · :9000", "Холодный архив: logs > 30 дней, vacancies > 7 дней")

    Container(prometheus, "Prometheus", "prom/prometheus · :9090", "Сбор метрик: HTTP latency, cache hit/miss, kafka publish, business counters")
    Container(grafana, "Grafana", "grafana/grafana · :3000", "Дашборды. Datasource Prometheus автоматически провизионирован")
  }

  System_Ext(hh_ru, "HH.ru")
  System_Ext(ollama, "Ollama")

  Rel(user, nginx, "HTTP :80", "JSON REST")

  Rel(nginx, auth_svc, "/auth/*", ":8082")
  Rel(nginx, aggregate_svc, "/api/vacancies /api/hh-token /api/scheduler", ":8080")
  Rel(nginx, autoapply_svc, "/api/autoapply /api/archive", ":8081")

  Rel(auth_svc, postgres, "users: INSERT, SELECT")
  Rel(aggregate_svc, postgres, "hh_tokens, vacancies: R/W")
  Rel(autoapply_svc, postgres, "requests, logs: R/W")

  Rel(autoapply_svc, redis, "Token + status cache")
  Rel(autoapply_svc, minio, "Archive JSON batches")

  Rel(autoapply_svc, kafka, "→ parse.requested, vacancy_input")
  Rel(aggregate_svc, kafka, "→ vacancies.parsed, token.updated")
  Rel(kafka, autoapply_svc, "← vacancies.parsed, vacancy_output, token.updated")
  Rel(kafka, generate_svc, "← vacancy_input")
  Rel(generate_svc, kafka, "→ vacancy_output")

  Rel(aggregate_svc, hh_ru, "Playwright Firefox")
  Rel(autoapply_svc, hh_ru, "Playwright Firefox")
  Rel(generate_svc, ollama, "POST /api/generate", "HTTP")

  Rel(prometheus, autoapply_svc, "scrape /metrics", ":8081")
  Rel(grafana, prometheus, "PromQL query", ":9090")
```

## Запуск

| Контейнер | Команда |
|---|---|
| Infra + Auth + Gateway + Observability | `docker-compose up -d` |
| Java aggregate service | `cd hh_aggregate_service && mvn spring-boot:run` |
| Go autoapply service | `cd hh_autoapply_service && go run cmd/main.go` |
| Python generate service | `python generate_service/generate.py` |

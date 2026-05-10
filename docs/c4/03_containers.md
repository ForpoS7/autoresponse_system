# Уровень 3 — Контейнеры (Containers)

> Все исполняемые единицы системы, хранилища и их коммуникационные протоколы.
> Каждый прямоугольник — отдельный процесс / контейнер с указанием технологии.

```mermaid
flowchart TD
    USER["👤 Соискатель\nHTTP :80"]

    %% ─── Уровень 1: Gateway ───────────────────────────────────────
    NGINX["🔀 nginx\n──────────────────\nnginx:alpine · Docker\nAPI Gateway\nRate limit: 5 rpm /auth\n          60 rpm /api\nLoad balancing: least_conn"]

    %% ─── Уровень 2: Сервисы ───────────────────────────────────────
    subgraph SERVICES["Сервисы приложения"]
        direction LR
        AUTH["🔐 auth_service\n──────────────\nGo 1.22 · Docker :8082\nJWT HS256, BCrypt\nPOST /register /login\nPOST /validate"]

        AGGR["📋 hh_aggregate_service\n──────────────\nJava 21 · Spring Boot 3.2.5\nHost :8080\nPlaywright Firefox\nKafka Producer"]

        APPLY["⚡ hh_autoapply_service\n──────────────\nGo 1.22 · Host :8081\nPlaywright Firefox\nKafka Producer+Consumer\nRedis · MinIO"]

        GEN["✍️ generate_service\n──────────────\nPython 3.11 · Host\nKafka Consumer\nOllama HTTP client"]
    end

    %% ─── Уровень 3: Брокер ────────────────────────────────────────
    KAFKA["📨 Apache Kafka\n──────────────────────────────\nKRaft mode 3.7 · Docker :9092\nТопики: parse.requested · vacancies.parsed\n        token.updated · vacancy_input · vacancy_output"]

    %% ─── Уровень 4: Хранилища ─────────────────────────────────────
    subgraph STORES["Хранилища данных"]
        direction LR
        PG["🐘 PostgreSQL 16\n──────────────\nDocker :5444\nusers · hh_tokens\nauto_apply_requests\nauto_apply_logs · vacancies"]

        REDIS["⚡ Redis 7\n──────────────\nDocker :6379\nhh_token:{userId} 24h TTL\nautoapply:{id} 1h TTL"]

        MINIO["🪣 MinIO\n──────────────\nDocker :9000\nS3-compatible\nХолодный архив\nlogs >30d, vac >7d"]
    end

    %% ─── Уровень 5: Observability ─────────────────────────────────
    subgraph OBS["Observability"]
        direction LR
        PROM["📊 Prometheus\n──────────\nDocker :9090\nscrape /metrics"]
        GRAF["📈 Grafana\n──────────\nDocker :3000\nDashboards"]
    end

    %% ─── Внешние ──────────────────────────────────────────────────
    HH["🌐 HH.ru"]
    OLLAMA["🤖 Ollama :11434"]

    %% ─── Связи: User → Gateway → Services ────────────────────────
    USER -->|"HTTP REST"| NGINX

    %% auth_request: nginx валидирует JWT через auth_service до проксирования
    %% auth_service возвращает X-User-ID → nginx передаёт в бэкенд
    %% сервисы JWT не трогают — только читают заголовок X-User-ID
    NGINX -->|"auth_request /validate\n(каждый /api/* запрос)"| AUTH
    NGINX -->|"/auth/register\n/auth/login\n:8082"| AUTH
    NGINX -->|"/api/vacancies\n/api/hh-token\n/api/scheduler\nX-User-ID header\n:8080"| AGGR
    NGINX -->|"/api/autoapply\n/api/archive\nX-User-ID header\n:8081"| APPLY

    %% ─── Services → Kafka ─────────────────────────────────────────
    APPLY -->|"→ parse.requested\n→ vacancy_input"| KAFKA
    AGGR  -->|"→ vacancies.parsed\n→ token.updated"| KAFKA
    KAFKA -->|"← vacancies.parsed\n← vacancy_output\n← token.updated"| APPLY
    KAFKA -->|"← vacancy_input"| GEN
    GEN   -->|"→ vacancy_output"| KAFKA

    %% ─── Services → Storage ───────────────────────────────────────
    AUTH  -->|"SELECT/INSERT users"| PG
    AGGR  -->|"R/W hh_tokens\nvacancies"| PG
    APPLY -->|"R/W requests\nlogs"| PG
    APPLY -->|"GET/SET\nJSON cache"| REDIS
    APPLY -->|"Upload\nJSON batches"| MINIO

    %% ─── Services → External ──────────────────────────────────────
    AGGR  -->|"Playwright\nFirefox"| HH
    APPLY -->|"Playwright\nFirefox"| HH
    GEN   -->|"POST /api/generate"| OLLAMA

    %% ─── Observability ────────────────────────────────────────────
    PROM  -->|"scrape :8081/metrics"| APPLY
    GRAF  -->|"PromQL :9090"| PROM
```

## Размещение контейнеров

| Контейнер | Рантайм | Порт | Запуск |
|---|---|---|---|
| nginx | Docker | :80 | `docker-compose up -d nginx` |
| auth_service | Docker | :8082 | `docker-compose up -d auth_service` |
| hh_aggregate_service | Host JVM | :8080 | `mvn spring-boot:run` |
| hh_autoapply_service | Host Go | :8081 | `go run cmd/main.go` |
| generate_service | Host Python | — | `python generate_service/generate.py` |
| PostgreSQL | Docker | :5444 | `docker-compose up -d postgres` |
| Kafka | Docker | :9092 | `docker-compose up -d kafka` |
| Redis | Docker | :6379 | `docker-compose up -d redis` |
| MinIO | Docker | :9000/:9001 | `docker-compose up -d minio` |
| Prometheus | Docker | :9090 | `docker-compose up -d prometheus` |
| Grafana | Docker | :3000 | `docker-compose up -d grafana` |

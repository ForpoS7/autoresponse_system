# Уровень 4 — Подконтейнеры (Subcontainers)

> Внутренние модули наиболее сложного контейнера — **hh_autoapply_service**.
> Каждый модуль — отдельный Go-пакет со своей зоной ответственности.

```mermaid
flowchart TD
    %% ─── Входящие связи ──────────────────────────────────────────
    NGINX["nginx :8081"]
    KAFKA_IN["Kafka\n← vacancies.parsed\n← vacancy_output\n← token.updated"]

    %% ─── Точка входа ─────────────────────────────────────────────
    CMD["cmd/main.go\n──────────────\nИнициализация:\n· конфиг (env)\n· PostgreSQL pool\n· Redis client\n· MinIO client\n· Kafka consumers\n· HTTP server + mux\n· Prometheus /metrics"]

    subgraph INTERNAL["internal/ — бизнес-логика"]
        direction TB

        subgraph LAYER_HTTP["HTTP Layer"]
            direction LR
            MW["middleware/\nmetrics.go\n──────────────\nPrometheus HTTP\nhttp_requests_total\nhttp_request_duration_seconds"]
            H_AA["handler/\nautoapply_handler.go\n──────────────\nPOST /api/autoapply\nGET  /api/autoapply/{id}\ncache-aside Redis"]
            H_TK["handler/\ntoken_handler.go\n──────────────\nGET  /api/hh-token\nPOST /api/hh-token"]
            H_AR["handler/\narchiver_handler.go\n──────────────\nPOST /api/archive/run\nGET  /api/archive/list"]
        end

        subgraph LAYER_SVC["Service Layer"]
            direction LR
            SVC_AA["service/\nautoapply_service.go\n──────────────\nEDA-оркестратор:\n1. getHHToken\n2. publish parse.requested\n3. waitForVacancies (2m)\n4. generateLetter × N\n5. applyToVacancy × N\n6. updateStatus"]
            SVC_AR["service/\narchiver_service.go\n──────────────\nArchiveLogs(>30d)\nArchiveVacancies(>7d)\nStartScheduler(24h)"]
        end

        subgraph LAYER_CACHE["Cache Layer"]
            direction LR
            TCACHE["cache/\ntoken_cache.go\n──────────────\nL1: sync.Map (in-memory)\nL2: Redis hh_token:{id}\nTTL: 24h\nGraceful degradation"]
        end
    end

    subgraph PKG["pkg/ — переиспользуемые пакеты"]
        direction TB

        subgraph PKG_KAFKA["pkg/kafka/"]
            direction LR
            PUB_P["parse_publisher.go\n→ parse.requested\n{jobId, query, page}"]
            CORR_V["vacancy_correlator.go\n← vacancies.parsed\nsync.Map[jobId]→chan"]
            PUB_L["letter_publisher.go\n→ vacancy_input\n{corrId, title, company}"]
            CORR_L["letter_correlator.go\n← vacancy_output\nsync.Map[corrId]→chan"]
            CONS_T["token_consumer.go\n← token.updated\n→ TokenCache.Set"]
        end

        subgraph PKG_OTHER["pkg/other"]
            direction LR
            REDIS_CL["redis/client.go\ngo-redis/v9\nGet/Set/GetJSON\nSetJSON/Del/Ping\nErrCacheMiss"]
            PLAYWRIGHT["playwright/\nautoapply.go\nLogin(storageState)\nApplyToVacancy(vacancy, letter)"]
            MINIO_CL["minio/client.go\nminio-go/v7\nUpload(key, data)\nList(prefix)"]
            METRICS["metrics/metrics.go\npromauto регистрация\nautoapply_requests_total\nkafka_messages_published\nredis_cache_hits/misses"]
        end
    end

    %% ─── Исходящие связи ─────────────────────────────────────────
    PG["PostgreSQL"]
    REDIS["Redis"]
    MINIO["MinIO"]
    KAFKA_OUT["Kafka\n→ parse.requested\n→ vacancy_input"]
    HH["HH.ru"]

    %% ─── Связи ────────────────────────────────────────────────────
    NGINX --> CMD
    CMD --> MW
    MW --> H_AA
    MW --> H_TK
    MW --> H_AR

    H_AA --> SVC_AA
    H_AR --> SVC_AR
    H_AA --> REDIS_CL

    SVC_AA --> TCACHE
    SVC_AA --> PUB_P
    SVC_AA --> CORR_V
    SVC_AA --> PUB_L
    SVC_AA --> CORR_L
    SVC_AA --> PLAYWRIGHT

    TCACHE --> REDIS_CL
    CONS_T --> TCACHE

    SVC_AR --> MINIO_CL

    KAFKA_IN --> CORR_V
    KAFKA_IN --> CORR_L
    KAFKA_IN --> CONS_T

    PUB_P --> KAFKA_OUT
    PUB_L --> KAFKA_OUT

    REDIS_CL --> REDIS
    MINIO_CL --> MINIO
    PLAYWRIGHT --> HH

    SVC_AA -.->|"SQL lib/pq"| PG
    SVC_AR -.->|"SQL lib/pq"| PG
    H_AA -.->|"SQL lib/pq"| PG
```

## Модульная структура пакетов

| Пакет | Ответственность | Внешние зависимости |
|---|---|---|
| `cmd/` | Точка входа, инициализация DI | все пакеты |
| `internal/middleware/` | HTTP-метрики Prometheus | `pkg/metrics` |
| `internal/handler/` | HTTP-хендлеры, routing | `internal/service`, `pkg/redis` |
| `internal/service/` | Бизнес-логика EDA | `pkg/kafka`, `pkg/playwright`, `internal/cache` |
| `internal/cache/` | Двухуровневый кеш токенов | `pkg/redis` |
| `pkg/kafka/` | Kafka producer/consumer + корреляция | `kafka-go` |
| `pkg/redis/` | Redis CRUD обёртка | `go-redis/v9` |
| `pkg/playwright/` | Playwright автоматизация HH.ru | `playwright-go` |
| `pkg/minio/` | MinIO S3 upload/list | `minio-go/v7` |
| `pkg/metrics/` | Prometheus метрики (promauto) | `prometheus/client_golang` |

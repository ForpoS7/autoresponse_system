# C4 Dynamic — Data Tier Flow

Как данные движутся между тремя уровнями хранения: Hot → Warm → Cold.

```mermaid
flowchart TD
  subgraph write["Запись данных (runtime)"]
    direction LR
    AA["POST /api/autoapply\nCreateAutoApplyRequest"]
    TK["POST /api/hh-token\nExtractHHToken"]
    LG["processAutoApply\ncreateAutoApplyLog"]
    VC["VacancyPublisher\nsaveVacancy"]
  end

  subgraph hot["🔥 Hot — PostgreSQL (всегда)"]
    direction TB
    users["users\nemail, password_hash"]
    hh_tokens["hh_tokens\nstorageState (Playwright)"]
    aa_req["auto_apply_requests\nstatus, applied_count"]
  end

  subgraph warm["🟡 Warm — PostgreSQL (с TTL)"]
    direction TB
    aa_logs["auto_apply_logs\n≤ 30 дней"]
    vacancies["vacancies\n≤ 7 дней"]
  end

  subgraph cache["⚡ Cache — Redis"]
    direction TB
    token_c["hh_token:{userId}\nTTL = 24h\nPlaywright storageState"]
    status_c["autoapply:{id}\nTTL = 1h (in-progress)\nTTL = ∞ (terminal)"]
  end

  subgraph cold["🧊 Cold — MinIO S3 (permanent)"]
    direction TB
    cold_logs["cold/logs/YYYY/MM/DD/\nbatch_N_ts.json"]
    cold_vacs["cold/vacancies/YYYY/MM/DD/\nbatch_N_ts.json"]
  end

  Archiver{{"ArchiverService\n⏱ каждые 24h\n📍 POST /api/archive/run"}}

  AA -->|"INSERT"| aa_req
  TK -->|"INSERT / UPDATE"| hh_tokens
  LG -->|"INSERT"| aa_logs
  VC -->|"INSERT"| vacancies

  hh_tokens -.->|"Kafka: token.updated\n→ TokenConsumer"| token_c
  aa_req -.->|"cache-aside\nGET /api/autoapply/{id}"| status_c

  aa_logs -->|"SELECT WHERE created_at < NOW() - 30d"| Archiver
  vacancies -->|"SELECT WHERE parsed_at < NOW() - 7d"| Archiver

  Archiver -->|"JSON marshal\nUpload(key, data)"| cold_logs
  Archiver -->|"JSON marshal\nUpload(key, data)"| cold_vacs
  Archiver -->|"DELETE WHERE id IN (...)"| aa_logs
  Archiver -->|"DELETE WHERE id IN (...)"| vacancies

  style hot fill:#fff3cd,stroke:#ffc107
  style warm fill:#d1ecf1,stroke:#17a2b8
  style cache fill:#d4edda,stroke:#28a745
  style cold fill:#cce5ff,stroke:#004085
  style write fill:#f8f9fa,stroke:#6c757d
```

## Таблица уровней

| Уровень | Хранилище | Таблицы / Пути | Срок жизни | Операции |
|---|---|---|---|---|
| **Hot** | PostgreSQL :5444 | `users`, `hh_tokens`, `auto_apply_requests` | Бессрочно | INSERT, SELECT, UPDATE |
| **Warm** | PostgreSQL :5444 | `auto_apply_logs`, `vacancies` | 30 / 7 дней | INSERT, SELECT, DELETE (ArchiverService) |
| **Cache** | Redis :6379 | `hh_token:{userId}`, `autoapply:{id}` | 24h / 1h / ∞ | GET, SET (TTL), DEL |
| **Cold** | MinIO :9000 | `cold/logs/…`, `cold/vacancies/…` | Постоянно | PUT (archive), GET (restore), LIST |

## Ключ архива в MinIO

```
cold/{type}/YYYY/MM/DD/batch_{count}_{unix_milli}.json
```

Пример: `cold/logs/2025/05/07/batch_142_1746619200000.json`

## Управление архивом

```bash
# Ручной запуск архивации
POST http://localhost/api/archive/run

# Список архивов
GET http://localhost/api/archive/list?prefix=logs
GET http://localhost/api/archive/list?prefix=vacancies

# MinIO Web Console
http://localhost:9001  (minioadmin / minioadmin)
```

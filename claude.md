# CLAUDE.md — Контекст проекта HH AutoResponse System

## О проекте
Учебный проект. Распределённая система для автопарсинга вакансий с HH.ru и автоматических откликов.
Написан двумя студентами параллельно — отсюда дублирование аутентификации и Playwright между Java и Go.

## Договорённости с пользователем
- **База данных**: изменения только концептуально (на словах), схему не трогаем
- **Упрощения**: это учебный проект, не усложнять без необходимости
- Предлагать изменения, не внедрять без явного запроса

---

## Актуальная архитектура

```
Client
   │
   ▼  :80
nginx (API Gateway)          ← docker-compose.yml (корень)
   │  rate limiting
   │  load balancing (least_conn)
   │
   ├──→ /auth/*        → auth_service     :8082  (Docker)
   ├──→ /api/vacancies → aggregate_service :8080  (хост, make java)
   ├──→ /api/hh-token  → aggregate_service :8080
   ├──→ /api/scheduler → aggregate_service :8080
   └──→ /api/autoapply → autoapply_service :8081  (хост, make go)

PostgreSQL :5444  (Docker, общая база — known issue)
Kafka      :9092  (Docker)
```

---

## Сервисы

### auth_service (Go, НОВЫЙ, порт 8082)
- Расположение: `auth_service/`
- Запуск в Docker: `docker-compose up -d auth_service`
- Endpoints:
  - POST `/register` → регистрация, возвращает JWT
  - POST `/login`    → вход, возвращает JWT
  - POST `/validate` → валидация токена (для других сервисов)
  - GET  `/health`
- JWT: HS256, secret совпадает с Java и Go сервисами
- Пишет в таблицу `users` (та же, что у Java/Go сервисов)

### hh_aggregate_service (Java/Spring Boot 3.2.5, порт 8080)
- Расположение: `hh_aggregate_service/`
- Запуск: `make java` (на хосте, не в Docker)
- Своя аутентификация — дублирует auth_service (known issue, не трогаем)
- Парсинг вакансий через Playwright (Firefox)
- Публикует в Kafka топик `vacancies.parsed`
- Планировщик — закомментирован

### hh_autoapply_service (Go 1.22, порт 8081)
- Расположение: `hh_autoapply_service/`
- Запуск: `make go` (на хосте, не в Docker)
- Своя аутентификация — дублирует auth_service (known issue, не трогаем)
- Автоотклики через Playwright
- Генерация писем: MockCoverLetterService (реальный Python сервис НЕ подключён)
- Получает HH-токен HTTP запросом к Java (tight coupling, known issue)

### generate_service (Python)
- Расположение: `generate_service/`
- Запуск: `python generate_service/generate.py`
- Kafka consumer: `vacancy_input` → Ollama (qwen2.5:7b) → `vacancy_output`
- НЕ интегрирован с Go сервисом (Go использует mock)

### nginx (API Gateway)
- Расположение: `nginx/nginx.conf`
- Запуск в Docker: `docker-compose up -d nginx`
- Rate limiting:
  - `/auth/*` → 5 req/min (защита от брутфорса)
  - `/api/*`  → 60 req/min
- Load balancer: алгоритм least_conn
- Для масштабирования: добавить строки `server host:port;` в upstream блоки

---

## Инфраструктура

### Docker Compose (корень проекта — docker-compose.yml)
Заменяет старый `hh_aggregate_service/docker-compose.yml`.
```
docker-compose up -d          # поднять всё (postgres + kafka + auth + nginx)
docker-compose up -d infra    # только postgres + kafka
docker-compose ps             # статус
docker-compose logs -f        # логи
```

### Kafka топики (EDA реализован)
| Топик | Продюсер | Консьюмер | Статус |
|-------|----------|-----------|--------|
| vacancies.parsed  | Java (VacanciesBatch с jobId) | Go VacancyCorrelator | ✅ с корреляцией |
| parse.requested   | Go ParsePublisher             | Java ParseRequestConsumer | ✅ новый |
| token.updated     | Java TokenPublisher           | Go TokenConsumer → TokenCache | ✅ новый |
| vacancy_input     | Go LetterPublisher            | Python generate_service | ✅ подключён |
| vacancy_output    | Python generate_service       | Go LetterCorrelator | ✅ подключён |

### EDA флоу (новый)
1. POST /api/autoapply → Go публикует parse.requested {jobId}
2. Java ParseRequestConsumer получает → парсит → публикует vacancies.parsed {jobId}
3. Go VacancyCorrelator маршрутизирует по jobId → горутина получает вакансии
4. Для каждой вакансии Go публикует vacancy_input {correlationId}
5. Python generate_service генерирует письмо → vacancy_output {correlationId}
6. Go LetterCorrelator маршрутизирует → письмо используется при отклике
7. HH-токен: Java публикует token.updated → Go TokenConsumer → TokenCache
   (нет HTTP Go→Java)

### Новые файлы (EDA)
**Java:** ParseRequestMessage, TokenUpdatedMessage, VacanciesBatch, KafkaConsumerConfig, ParseRequestConsumer, TokenPublisher
**Go:** pkg/kafka/{topics,messages,parse_publisher,vacancy_correlator,letter_publisher,letter_correlator,token_consumer}.go + internal/cache/token_cache.go

---

## Хранение данных (реализовано)

### Три уровня
| Уровень | Хранилище | Данные | Срок |
|---------|-----------|--------|------|
| Hot | PostgreSQL | users, hh_tokens, auto_apply_requests (активные) | всегда |
| Warm | PostgreSQL | auto_apply_logs, vacancies | до 30 / 7 дней |
| Cold | MinIO :9000 | archived logs + vacancies (JSON батчи) | постоянно |

### Архивация
- `ArchiverService` читает старые записи из PostgreSQL → сериализует в JSON → загружает в MinIO → удаляет из PostgreSQL
- Путь в MinIO: `cold/{logs|vacancies}/YYYY/MM/DD/batch_{count}_{ts}.json`
- Scheduler: автоматически каждые `archive_interval_hours` часов
- Ручной запуск: `POST /api/archive/run`
- Список архивов: `GET /api/archive/list?prefix=logs`
- MinIO console: http://localhost:9001 (minioadmin/minioadmin)

### Кеш Redis (реализован, порт 6379)
| Ключ | TTL | Что хранится |
|------|-----|--------------|
| `hh_token:{userId}` | 24h | HH storageState (Playwright session) |
| `autoapply:{id}` | 1h / ∞ terminal | AutoApplyResponse для GET /api/autoapply/{id} |

- `pkg/redis/client.go` — обёртка над go-redis/v9: Get/Set/GetJSON/SetJSON/Del/Ping
- `internal/cache/token_cache.go` — Redis primary + in-memory fallback (graceful degradation)
- `internal/handler/autoapply_handler.go` — cache-aside на GET статуса; исправлен баг r.PathValue → mux.Vars
- `cmd/main.go` — инициализирует Redis, graceful skip если недоступен
- `go.mod` — добавлен github.com/redis/go-redis/v9 v9.5.1

### Observability (реализован)

**Стек:** Prometheus :9090 + Grafana :3000

**Метрики (Go autoapply_service):**
| Метрика | Тип | Лейблы |
|---------|-----|--------|
| `http_requests_total` | counter | method, path, status |
| `http_request_duration_seconds` | histogram | method, path |
| `autoapply_requests_total` | counter | status (created/completed/failed) |
| `kafka_messages_published_total` | counter | topic |
| `redis_cache_hits_total` | counter | cache (token/autoapply) |
| `redis_cache_misses_total` | counter | cache (token/autoapply) |

**Новые файлы:**
- `pkg/metrics/metrics.go` — глобальные Prometheus метрики (promauto)
- `internal/middleware/metrics.go` — HTTP middleware (gorilla/mux route template как label, избегает high cardinality)
- `prometheus/prometheus.yml` — scrape config: autoapply :8081/metrics + self
- `grafana/provisioning/datasources/prometheus.yml` — автоматический datasource Grafana

**Изменённые файлы:**
- `cmd/main.go` — `r.Use(middleware.Metrics)` + `/metrics` endpoint (promhttp)
- `internal/handler/autoapply_handler.go` — инкремент `autoapply_requests_total` + cache hit/miss
- `internal/cache/token_cache.go` — cache hit/miss метрики
- `docker-compose.yml` — prometheus + grafana сервисы; исправлен redis_data volume

**Доступ:**
- Prometheus UI: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)
- Raw metrics: http://localhost:8081/metrics

## Известные проблемы (не трогать без явного запроса)
1. Дублирование аутентификации в Java и Go (теперь есть auth_service, но старая не удалена)
2. Общая PostgreSQL база для всех сервисов (концептуально — разные БД, но не реализовано)
3. generate_service не интегрирован с Go (используется Mock)
4. HTTP Go→Java для HH-токена ~~(tight coupling, нет circuit breaker)~~ → решено через token.updated Kafka + TokenCache
5. Планировщик в Java закомментирован
6. JWT secret захардкожен в конфигах

---

## Среда
- Windows 11, PowerShell
- `make` не установлен — запускать команды вручную через Git Bash
- Docker Desktop нужен для `docker-compose`
- Java сервис и Go сервис запускаются на хосте (не в Docker)
- nginx обращается к ним через `host.docker.internal`

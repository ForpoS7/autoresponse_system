# HH AutoApply Service — Specification

## Общая информация

| Параметр | Значение |
|----------|----------|
| **Язык** | Go 1.22+ |
| **Фреймворк** | Gorilla Mux |
| **Порт** | 8081 |
| **Основное назначение** | Автоотклики на вакансии с использованием Playwright |
| **Build tool** | Go modules |

## Функционал

### 1. Аутентификация и авторизация

- **Регистрация пользователей** (`POST /api/auth/register`)
- **Логин** (`POST /api/auth/login`)
- **JWT токены** с expiration 24 часа
- **Кастомная JWT реализация** на основе `github.com/golang-jwt/jwt/v5`

### 2. Парсинг вакансий

- **Автоматический парсинг** через Playwright
- **Извлечение данных:**
  - ID вакансии
  - Название позиции
  - Работодатель
  - URL вакансии
  - Описание (полный текст)
  - Зарплатная вилка (from, to)
  - Валюта
  - Регион
- **Публикация в Kafka** топик `vacancies.parsed`

### 3. Автоотклики

- **Создание запроса на автоотклик** (`POST /api/autoapply`)
- **Отслеживание статуса** (`GET /api/autoapply/{id}`)
- **Автоматическая генерация сопроводительных писем** через generate_service (Python + Ollama)
- **Логирование результатов** откликов

### 4. Управление токеном HH.ru

- **Извлечение токена** из cookies HH.ru через браузер
- **Сохранение токена** в базу данных
- **Получение сохранённого токена**

### 5. Интеграция с Java сервисом

- **HTTP клиент** для взаимодействия с `hh_aggregate_service`
- **Получение токена** из Java сервиса для межсервисной авторизации

## Структура проекта

```
hh_autoapply_service/
├── cmd/
│   └── main.go                           # Точка входа, DI контейнер
├── internal/
│   ├── config/
│   │   └── config.go                     # Загрузка конфигурации из YAML
│   ├── handler/
│   │   ├── auth_handler.go               # Auth endpoints
│   │   ├── token_handler.go              # HH token endpoints
│   │   └── autoapply_handler.go          # AutoApply endpoints
│   ├── jwt/
│   │   └── jwt.go                        # JWT утилиты
│   ├── middleware/
│   │   └── auth_middleware.go            # JWT middleware
│   ├── model/
│   │   ├── user.go                       # User модель
│   │   ├── hh_token.go                   # HHToken модель
│   │   ├── vacancy.go                    # Vacancy модель
│   │   └── autoapply.go                  # AutoApplyRequest модель
│   ├── repository/
│   │   ├── user_repository.go            # CRUD users
│   │   ├── hh_token_repository.go        # CRUD hh_tokens
│   │   ├── vacancy_repository.go         # CRUD vacancies
│   │   ├── autoapply_repository.go       # CRUD auto_apply_requests
│   │   └── schema.sql                    # SQL схема БД
│   └── service/
│       ├── auth_service.go               # Логика авторизации
│       ├── token_service.go              # Управление HH токеном
│       ├── parser_service.go             # Парсинг вакансий
│       ├── playwright_service.go         # Работа с браузером
│       └── autoapply_service.go          # Логика автооткликов
├── pkg/
│   ├── ai/
│   │   └── cover_letter_service.go         # Интеграция с generate_service (Kafka)
│   ├── httpclient/
│   │   └── hh_aggregate_client.go          # HTTP клиент к Java сервису
│   ├── kafka/
│   │   ├── producer.go                     # Kafka продюсер
│   │   └── consumer.go                     # Kafka консьюмер
│   ├── playwright/
│   │   └── browser_manager.go              # Управление браузером
│   └── ratelimit/
│       └── rate_limiter.go                 # Rate limiting
├── config.yml                            # Конфигурация приложения
├── docker-compose.yml                    # Docker инфраструктура
├── go.mod                                # Go модуль
├── go.sum                                # Go зависимости
└── README.md                             # Документация
```

## Зависимости (go.mod)

### Основные

```go
github.com/gorilla/mux v1.8.1              // HTTP роутер
github.com/lib/pq v1.10.9                  // PostgreSQL драйвер
github.com/golang-jwt/jwt/v5 v5.2.0        // JWT токены
github.com/segmentio/kafka-go v0.4.47      // Kafka клиент
github.com/playwright-community/playwright-go v0.5700.1  // Browser automation
golang.org/x/crypto v0.19.0                // Хеширование паролей
golang.org/x/time v0.5.0                   // Rate limiting
gopkg.in/yaml.v3 v3.0.1                    // YAML парсинг
```

## API Endpoints

### Аутентификация

| Метод | Endpoint | Описание | Auth |
|-------|----------|----------|------|
| POST | `/api/auth/register` | Регистрация нового пользователя | ❌ |
| POST | `/api/auth/login` | Логин, получение JWT токена | ❌ |

### Парсинг

| Метод | Endpoint | Описание | Auth |
|-------|----------|----------|------|
| GET | `/api/vacancies?query=&page=` | Парсинг вакансий с HH.ru | ✅ |

### Токен HH.ru

| Метод | Endpoint | Описание | Auth |
|-------|----------|----------|------|
| POST | `/api/hh-token` | Извлечение токена из cookies HH.ru | ✅ |
| GET | `/api/hh-token` | Получение сохранённого токена | ✅ |

### Автоотклик

| Метод | Endpoint | Описание | Auth |
|-------|----------|----------|------|
| POST | `/api/autoapply` | Создать запрос на автоотклик | ✅ |
| GET | `/api/autoapply/{id}` | Получить статус автоотклика | ✅ |

### Планировщик

| Метод | Endpoint | Описание | Auth |
|-------|----------|----------|------|
| GET | `/api/scheduler/config` | Конфигурация планировщика | ✅ |

### Health Check

| Метод | Endpoint | Описание | Auth |
|-------|----------|----------|------|
| GET | `/health` | Проверка доступности сервиса | ❌ |

## База данных

### Схема (internal/repository/schema.sql)

#### users
| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| email | VARCHAR(255) | UNIQUE, NOT NULL |
| password_hash | VARCHAR(255) | NOT NULL |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT CURRENT_TIMESTAMP |
| updated_at | TIMESTAMP WITH TIME ZONE | DEFAULT CURRENT_TIMESTAMP |

#### hh_tokens
| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| user_id | BIGINT | FOREIGN KEY → users(id), ON DELETE CASCADE |
| token_value | TEXT | NOT NULL |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT CURRENT_TIMESTAMP |
| updated_at | TIMESTAMP WITH TIME ZONE | DEFAULT CURRENT_TIMESTAMP |

#### vacancies
| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| title | VARCHAR(500) | NOT NULL |
| employer | VARCHAR(255) | - |
| url | VARCHAR(500) | NOT NULL |
| description | TEXT | - |
| salary_from | BIGINT | - |
| salary_to | BIGINT | - |
| currency | VARCHAR(10) | - |
| region | VARCHAR(255) | - |
| user_id | BIGINT | FOREIGN KEY → users(id), ON DELETE CASCADE |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT CURRENT_TIMESTAMP |

#### auto_apply_requests
| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| user_id | BIGINT | FOREIGN KEY → users(id), ON DELETE CASCADE |
| query | VARCHAR(500) | NOT NULL |
| apply_count | INT | NOT NULL DEFAULT 0 |
| applied_count | INT | NOT NULL DEFAULT 0 |
| status | VARCHAR(50) | NOT NULL DEFAULT 'pending' |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT CURRENT_TIMESTAMP |
| updated_at | TIMESTAMP WITH TIME ZONE | DEFAULT CURRENT_TIMESTAMP |

#### auto_apply_logs
| Column | Type | Constraints |
|--------|------|-------------|
| id | BIGSERIAL | PRIMARY KEY |
| request_id | BIGINT | FOREIGN KEY → auto_apply_requests(id), ON DELETE CASCADE |
| vacancy_id | BIGINT | NOT NULL |
| vacancy_url | VARCHAR(500) | NOT NULL |
| cover_letter | TEXT | - |
| status | VARCHAR(50) | NOT NULL |
| error_message | TEXT | - |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT CURRENT_TIMESTAMP |

### Индексы

```sql
CREATE INDEX idx_hh_tokens_user_id ON hh_tokens(user_id);
CREATE INDEX idx_vacancies_user_id ON vacancies(user_id);
CREATE INDEX idx_auto_apply_requests_user_id ON auto_apply_requests(user_id);
CREATE INDEX idx_auto_apply_logs_request_id ON auto_apply_logs(request_id);
```

## Kafka

### Консьюмер: AutoApply Service

**Топик:** `vacancies.parsed`  
**Group ID:** `autoapply-service-group`

**Потребитель:**
- Получает распарсенные вакансии из Kafka
- Обрабатывает вакансии для автооткликов
- Сохраняет результаты в базу данных

### Продюсер: VacancyPublisher

**Топик:** `vacancies.parsed`

**Формат сообщения:**
```json
[
  {
    "id": 12345678,
    "title": "Java Developer",
    "employer": "Яндекс",
    "url": "https://hh.ru/vacancy/12345678",
    "description": "Разработка backend-сервисов...",
    "salaryFrom": 150000,
    "salaryTo": 250000,
    "currency": "RUR",
    "region": "Москва"
  }
]
```

### Интеграция с Generate Service

**Топики:**
- `vacancy_input` — запрос на генерацию сопроводительного письма
- `vacancy_output` — сгенерированное письмо

**Поток:**
1. Go сервис отправляет вакансию в `vacancy_input`
2. Python сервис генерирует письмо через Ollama
3. Python сервис публикует результат в `vacancy_output`
4. Go сервис получает письмо и использует для автоотклика

## Конфигурация (config.yml)

```yaml
server:
  port: 8081

database:
  host: localhost
  port: 5444
  name: hh
  user: postgres
  password: postgres
  sslmode: disable

kafka:
  brokers:
    - localhost:9092
  topic:
    vacancies: vacancies.parsed

jwt:
  secret: defaultSecretKeyForDevelopment...
  expiration: 86400000  # 24 часа
  java_service_token: eyJhbGciOiJIUzI1NiJ9...  # Токен для межсервисной авторизации

playwright:
  headless: false  # Показывать браузер
  area_code: 1     # Москва
  slow_mo: 100     # Задержка для отладки

rate_limiter:
  enabled: true
  requests_per_minute: 10
  burst: 5

hh:
  api_url: https://hh.ru

scheduler:
  parser:
    cron: ""  # Отключено
```

## Запуск

### Предварительная подготовка

```bash
# Установка браузеров Playwright
cd hh_autoapply_service
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install
```

### Через go run

```bash
cd hh_autoapply_service
go run cmd/main.go
```

### Сборка бинарника

```bash
cd hh_autoapply_service
go build -o bin/hh_autoapply_service ./cmd/main.go
./bin/hh_autoapply_service
```

## Проверка работы

```bash
# Health check
curl http://localhost:8081/health

# Регистрация
curl -X POST http://localhost:8081/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}'

# Автоотклик
curl -X POST http://localhost:8081/api/autoapply \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query": "Java Developer", "apply_count": 5}'
```

## Межсервисное взаимодействие

```
┌─────────────────────┐         ┌─────────────────────┐
│  hh_autoapply_svc   │  HTTP   │  hh_aggregate_svc   │
│  (Go, port 8081)    │ ──────> │  (Java, port 8080)  │
│                     │         │                     │
│  • Получение токена │         │  • Предоставление   │
│    для авторизации  │         │    токена           │
│  • Использование    │         │  • Валидация        │
│    Java Service     │         │    запросов         │
│    Token            │         │                     │
└─────────────────────┘         └─────────────────────┘
```

---

**Дата создания:** 2026-03-15  
**Версия спецификации:** 1.0

# HH AutoApply Service

Микросервис для автоматического отклика на вакансии HH.ru с использованием Playwright.

## Запуск

```bash
# Инфраструктура (PostgreSQL + Kafka)
docker-compose up -d

# Приложение
go run cmd/main.go
```

## API Endpoints

### Аутентификация

#### `POST /api/auth/register`
Регистрация пользователя.

**Запрос:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Ответ:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiJ9...",
  "expiresAt": "2026-03-12T15:20:00Z"
}
```

---

#### `POST /api/auth/login`
Вход (возвращает JWT токен).

**Запрос:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Ответ:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiJ9...",
  "expiresAt": "2026-03-12T15:20:00Z"
}
```

---

### Вакансии

#### `GET /api/vacancies?query=&page=`
Парсинг вакансий с HH.ru.

**Запрос:**
```http
GET /api/vacancies?query=Java Developer&page=0
Authorization: Bearer {token}
```

**Ответ:**
```json
[
  {
    "id": 1,
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

> ✅ Отправляет в Kafka: **`vacancies.parsed`**

---

### Токен HH.ru

#### `POST /api/hh-token`
Извлечение токена из cookies HH.ru (автоматически через браузер).

**Запрос:**
```http
POST /api/hh-token
Authorization: Bearer {token}
```

**Ответ:** `200 OK`

---

#### `GET /api/hh-token`
Получение сохранённого токена.

**Запрос:**
```http
GET /api/hh-token
Authorization: Bearer {token}
```

**Ответ:**
```json
{
  "tokenValue": "hh_token_value_here"
}
```

---

### Автоотклик

#### `POST /api/autoapply`
Создать запрос на автоматический отклик.

**Запрос:**
```json
{
  "query": "Java Developer",
  "apply_count": 10
}
```

**Ответ:**
```json
{
  "request_id": 1,
  "status": "pending",
  "message": "Auto-apply process started",
  "applied_count": 0
}
```

---

#### `GET /api/autoapply/{id}`
Получить статус запроса на автоотклик.

**Запрос:**
```http
GET /api/autoapply/1
Authorization: Bearer {token}
```

**Ответ:**
```json
{
  "request_id": 1,
  "status": "processing",
  "message": "",
  "applied_count": 5,
  "failed_count": 0
}
```

---

### Планировщик

#### `GET /api/scheduler/config`
Получить конфигурацию планировщика.

**Запрос:**
```http
GET /api/scheduler/config
Authorization: Bearer {token}
```

**Ответ:**
```json
{
  "parserCron": ""
}
```

---

## Kafka

| Топик | Описание                                  |
|-------|-------------------------------------------|
| `vacancies.parsed` | Сюда публикуются все распарсенные вакансии |

**Формат сообщения:**
```json
[
  {
    "id": 1,
    "title": "Java Developer",
    "employer": "Яндекс",
    "url": "https://hh.ru/vacancy/12345678",
    "description": "...",
    "salaryFrom": 150000,
    "salaryTo": 250000,
    "currency": "RUR",
    "region": "Москва"
  }
]
```

---

## Конфигурация

Конфигурационный файл: `config.yml`

```yaml
server:
  port: 8080

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
  expiration: 86400000 # 24 часа

playwright:
  headless: false
  area_code: 1 # Москва
  slow_mo: 100

rate_limiter:
  enabled: true
  requests_per_minute: 10
  burst: 5

scheduler:
  parser:
    cron: "" # Пустое значение отключает планировщик
```

## Health Check

```http
GET /health
```

Ответ: `OK`

## Структура проекта

```
hh_autoapply_service/
├── cmd/
│   └── main.go              # Точка входа
├── internal/
│   ├── config/              # Конфигурация
│   ├── handler/             # HTTP хендлеры
│   ├── jwt/                 # JWT утилиты
│   ├── middleware/          # HTTP middleware
│   ├── model/               # Модели данных
│   ├── repository/          # Репозитории (БД)
│   └── service/             # Бизнес-логика
├── pkg/
│   ├── ai/                  # AI сервис (мок)
│   ├── kafka/               # Kafka продюсер
│   ├── playwright/          # Playwright утилиты
│   └── ratelimit/           # Rate limiter
├── config.yml               # Конфигурация
├── docker-compose.yml       # Docker инфраструктура
└── go.mod                   # Go модуль
```

## Примечания

1. **Первый запуск:** Перед первым использованием необходимо зайти на hh.ru через браузер и авторизоваться, затем вызвать `POST /api/hh-token` для сохранения сессии.

2. **AI сопроводительные письма:** В текущей версии используется мок-сервис. Для интеграции реального AI необходимо заменить `MockCoverLetterService` на реальную реализацию.

3. **Rate Limiting:** По умолчанию установлен лимит 10 запросов в минуту с burst 5. Настройте под свои нужды в `config.yml`.

4. **Порт сервиса:** 8080 (как в hh_aggregate_service)

5. **Порт БД:** 5444 (как в hh_aggregate_service)

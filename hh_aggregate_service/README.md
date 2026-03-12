# HH Aggregate Service

Микросервис для парсинга вакансий HH.ru.

## Запуск

```bash
# Инфраструктура (PostgreSQL + Kafka)
docker-compose up -d

# Приложение
gradle bootRun
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

### Планировщик (В РАЗРАБОТКЕ - ПОКА РУЧКАМИ ДЕРГАЕТСЯ ПАРСИНГ)

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
  "parserCron": "0 0 */2 * * *"
}
```

---

## Kafka

| Топик | Описание                                  |
|-------|-------------------------------------------|
| `vacancies.parsed` | Сюда публикуются все распаршеные вакансии |

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

# HH Autoresponse System - Aggregate Service

Spring Boot микросервис для парсинга вакансий HH.ru и публикации их в Kafka.

## Архитектура

```
┌─────────────┐     ┌─────────────────────┐     ┌─────────────┐
│   Client    │────▶│  HH Aggregate       │────▶│   Kafka     │
│  (REST API) │     │  Service (8080)     │     │ (vacancies) │
└─────────────┘     └─────────────────────┘     └─────────────┘
                          │
                          ▼
                   ┌─────────────┐
                   │  PostgreSQL │
                   │   (5444)    │
                   └─────────────┘
```

## Технологии

- **Java 17+**
- **Spring Boot 3.2.5**
- **Spring Security + JWT**
- **Spring Data JPA**
- **Spring Kafka**
- **Playwright** (парсинг HH.ru)
- **PostgreSQL 16**
- **Apache Kafka 3.7.0**

## Структура проекта

```
hh_aggregate_service/
├── src/main/java/by/icemens/hh_aggregate_service/
│   ├── config/                    # Конфигурация безопасности, CORS
│   ├── controller/                # REST контроллеры
│   │   ├── AuthController.java    # /api/auth (login, register)
│   │   ├── ParserController.java  # /api/vacancies, /api/hh-token
│   │   └── SchedulerController.java # Управление планировщиком
│   ├── dto/                       # Data Transfer Objects
│   ├── entity/                    # JPA сущности
│   │   ├── User.java
│   │   ├── HhToken.java
│   │   └── Vacancy.java
│   ├── message/                   # Kafka сообщения
│   │   └── VacancyMessage.java
│   ├── publish/                   # Kafka продюсеры
│   │   └── VacancyPublisher.java
│   ├── repository/                # Spring Data репозитории
│   ├── security/                  # JWT фильтры, конфиги
│   └── service/                   # Бизнес-логика
│       ├── AuthService.java
│       ├── ParserService.java
│       ├── PlaywrightService.java
│       ├── TokenService.java
│       └── SchedulerService.java
├── src/main/resources/
│   └── application.yml            # Конфигурация приложения
├── docker-compose.yml             # PostgreSQL + Kafka
├── build.gradle
└── README.md
```

## Быстрый старт

### 1. Требования

- Java 17 или выше
- Docker и Docker Compose
- Gradle 8.x

### 2. Запуск инфраструктуры

```bash
cd hh_aggregate_service
docker-compose up -d
```

Запустятся:
- **PostgreSQL** на порту `5444`
- **Kafka** на порту `9092`

### 3. Запуск приложения

```bash
gradle bootRun
```

Или собрать JAR и запустить:

```bash
gradle build
java -jar build/libs/hh-aggregate-service-1.0-SNAPSHOT.jar
```

## API Endpoints

### Аутентификация

#### Регистрация
```http
POST /api/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

#### Вход
```http
POST /api/auth/login
Content-Type: application/json

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

### Парсинг вакансий

#### Получить вакансии
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
    "description": "...",
    "salaryFrom": 150000,
    "salaryTo": 250000,
    "currency": "RUR",
    "region": "Москва"
  }
]
```

#### Сохранить токен HH.ru
```http
POST /api/hh-token
Authorization: Bearer {token}
Content-Type: application/json

{
  "tokenValue": "hh_token_value"
}
```

### Планировщик

#### Включить/выключить парсинг по расписанию
```http
POST /api/scheduler/enable
Authorization: Bearer {token}
```

```http
POST /api/scheduler/disable
Authorization: Bearer {token}
```

## Конфигурация

## Kafka

Приложение публикует вакансии в топик **`vacancies.parsed`**.

Топик создаётся автоматически при первой отправке сообщения (настройка `KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"`).

### Формат сообщения

```json
{
  "userId": 1,
  "vacancyId": 12345,
  "title": "Java Developer",
  "employer": "Яндекс",
  "url": "https://hh.ru/vacancy/12345678",
  "salaryFrom": 150000,
  "salaryTo": 250000,
  "currency": "RUR",
  "region": "Москва"
}
```

## Безопасность

- Пароли хешируются (BCrypt)
- JWT токены для аутентификации
- CORS настроен для localhost

## База данных

Таблицы создаются автоматически (`ddl-auto: update`):

- `users` — пользователи
- `hh_tokens` — токены HH.ru
- `vacancies` — распарсенные вакансии

## Логирование

Уровень логирования в `application.yml`:

```yaml
logging:
  level:
    root: INFO
    by.icemens.hh_aggregate_service: DEBUG
    com.microsoft.playwright: WARN
```

## Troubleshooting

### Kafka: UNKNOWN_TOPIC_OR_PARTITION

Топик `vacancies.parsed` не существует. Проверьте, что в `docker-compose.yml`:

```yaml
KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"
```

И перезапустите Kafka:

```bash
docker-compose restart kafka
```

### Playwright: браузер не найден

```
Exception: Executable doesn't exist
```

Установите браузеры Playwright:

```bash
# Windows (PowerShell)
pwsh bin/install-playwright.ps1

# Linux/macOS
npx playwright install chromium
```

### Session expired на HH.ru

Обновите токен HH.ru через API:

```http
POST /api/hh-token
Authorization: Bearer {token}
Content-Type: application/json

{
  "tokenValue": "новый_токен"
}
```

## Тестирование

```bash
gradle test
```

## Сборка

```bash
gradle clean build
```

JAR файл: `build/libs/hh-aggregate-service-1.0-SNAPSHOT.jar`

## Лицензия

MIT

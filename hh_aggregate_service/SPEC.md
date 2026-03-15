# HH Aggregate Service — Specification

## Общая информация

| Параметр | Значение |
|----------|----------|
| **Язык** | Java 17+ |
| **Фреймворк** | Spring Boot 3.2.5 |
| **Порт** | 8080 |
| **Основное назначение** | Парсинг вакансий с HH.ru, публикация в Kafka |
| **Build tool** | Gradle 8.x |

## Функционал

### 1. Аутентификация и авторизация

- **Регистрация пользователей** (`POST /api/auth/register`)
- **Логин** (`POST /api/auth/login`)
- **JWT токены** с expiration 24 часа
- **Spring Security** с кастомным JWT фильтром

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

### 3. Управление токеном HH.ru

- **Извлечение токена** из cookies HH.ru через браузер
- **Сохранение токена** в базу данных
- **Получение сохранённого токена**

### 4. Планировщик (Scheduler)

- **Cron-задачи** для автоматического парсинга
- **Настраиваемый интервал** через application.yml
- **Отключён по умолчанию** (закомментирован в коде)

## Структура проекта

```
hh_aggregate_service/
├── src/main/
│   ├── java/by/icemens/hh_aggregate_service/
│   │   ├── HhAggregateServiceApplication.java    # Точка входа
│   │   ├── config/
│   │   │   ├── PlaywrightConfig.java             # Конфигурация Playwright
│   │   │   └── SecurityConfig.java               # Настройки безопасности
│   │   ├── controller/
│   │   │   ├── AuthController.java               # Auth endpoints
│   │   │   ├── ParserController.java             # Парсинг вакансий
│   │   │   ├── SchedulerController.java          # Scheduler config
│   │   │   └── TokenController.java              # HH token endpoints
│   │   ├── dto/
│   │   │   ├── AuthResponse.java                 # Ответ авторизации
│   │   │   ├── LoginRequest.java                 # Запрос логина
│   │   │   ├── ParserRequest.java                # Запрос парсинга
│   │   │   ├── RegisterRequest.java              # Запрос регистрации
│   │   │   ├── TokenRequest.java                 # Запрос токена
│   │   │   └── Vacancy.java                      # DTO вакансии
│   │   ├── entity/
│   │   │   ├── User.java                         # Entity пользователя
│   │   │   └── HhToken.java                      # Entity HH токена
│   │   ├── message/
│   │   │   └── VacancyMessage.java               # Kafka сообщение
│   │   ├── publish/
│   │   │   └── VacancyPublisher.java             # Публикация в Kafka
│   │   ├── repository/
│   │   │   ├── UserRepository.java               # Репозиторий users
│   │   │   └── HhTokenRepository.java            # Репозиторий hh_tokens
│   │   ├── security/
│   │   │   ├── JwtAuthenticationFilter.java      # JWT фильтр
│   │   │   └── JwtTokenProvider.java             # JWT утилиты
│   │   └── service/
│   │       ├── AuthService.java                  # Логика авторизации
│   │       ├── CustomUserDetailsService.java     # UserDetailsService
│   │       ├── ParserService.java                # Парсинг вакансий
│   │       ├── PlaywrightService.java            # Работа с браузером
│   │       ├── SchedulerService.java             # Cron задачи
│   │       └── TokenService.java                 # Управление HH токеном
│   └── resources/
│       └── application.yml                       # Конфигурация
├── build.gradle                                  # Gradle конфигурация
├── docker-compose.yml                            # Docker инфраструктура
├── gradle.properties                             # Gradle свойства
├── gradlew, gradlew.bat                          # Gradle wrapper
└── settings.gradle                               # Gradle настройки
```

## Зависимости (build.gradle)

### Основные

```gradle
spring-boot-starter-web          // REST API
spring-boot-starter-data-jpa     // JPA/Hibernate
spring-boot-starter-security     // Безопасность
spring-boot-starter-validation   // Валидация данных
spring-kafka                     // Kafka интеграция
postgresql                       // PostgreSQL драйвер
```

### Безопасность

```gradle
jjwt-api:0.12.5                  // JWT создание
jjwt-impl:0.12.5                 // JWT реализация
jjwt-jackson:0.12.5              // JWT JSON сериализация
```

### Утилиты

```gradle
lombok                           // Boilerplate reduction
playwright:1.58.0                // Browser automation
jsoup:1.18.1                     // HTML parsing/cleaning
jackson-databind                 // JSON processing
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

### Планировщик

| Метод | Endpoint | Описание | Auth |
|-------|----------|----------|------|
| GET | `/api/scheduler/config` | Конфигурация планировщика | ✅ |

### Health Check

| Метод | Endpoint | Описание | Auth |
|-------|----------|----------|------|
| GET | `/health` | Проверка доступности сервиса | ❌ |

## База данных

### Таблицы

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

**Индексы:**
- `idx_hh_tokens_user_id` ON `hh_tokens(user_id)`

## Kafka

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

## Конфигурация (application.yml)

```yaml
server:
  port: 8080

spring:
  datasource:
    url: jdbc:postgresql://localhost:5444/hh
    username: postgres
    password: postgres
  jpa:
    hibernate:
      ddl-auto: update

kafka:
  bootstrap-servers: localhost:9092

jwt:
  secret: defaultSecretKey...
  expiration: 86400000

playwright:
  headless: false
  area-code: 1  # Москва

scheduler:
  parser:
    cron: ""  # Отключено
```

## Запуск

### Через Gradle

```bash
cd hh_aggregate_service
./gradlew bootRun
```

### Через IDE

1. Открыть проект в IntelliJ IDEA
2. Найти `HhAggregateServiceApplication`
3. Run → Run 'HhAggregateServiceApplication'

## Проверка работы

```bash
# Health check
curl http://localhost:8080/health

# Регистрация
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}'
```

---

**Дата создания:** 2026-03-15  
**Версия спецификации:** 1.0

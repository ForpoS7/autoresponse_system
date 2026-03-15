# Инструкция по запуску HH AutoResponse System

## Требования

- **Go** 1.21+ (`go version`)
- **Java** 17+ (`java -version`)
- **Docker Desktop** (последняя версия)
- **Gradle** (для Java сервиса)

---

## Шаг 1: Проверка требований

```bash
# Проверка Go
go version
# Должно быть: go version go1.21.x или выше

# Проверка Java
java -version
# Должно быть: Java 17 или выше

# Проверка Docker
docker --version
docker-compose --version

# Проверка Gradle
gradle --version
# Если нет: brew install gradle
```

---

## Шаг 2: Запуск инфраструктуры (PostgreSQL + Kafka)

```bash
# Переходим в директорию с docker-compose
cd /Users/timofey/Programming/autoresponse_system/hh_aggregate_service

# Запускаем контейнеры
docker-compose up -d

# Проверяем статус
docker-compose ps

# Должны быть запущены:
# - hh-autoresponse-db (PostgreSQL на порту 5444)
# - hh-autoresponse-kafka (Kafka на порту 9092)
```

### Если контейнеры не запускаются:

```bash
# Проверка логов Docker
docker-compose logs

# Перезапуск контейнеров
docker-compose down
docker-compose up -d

# Проверка, что PostgreSQL доступен
docker-compose exec postgres pg_isready -U postgres
```

---

## Шаг 3: Запуск Java сервиса (hh_aggregate_service)

### Вариант A: Через Gradle

```bash
cd /Users/timofey/Programming/autoresponse_system/hh_aggregate_service

# Запуск
gradle bootRun

# Или через Gradle wrapper
./gradlew bootRun
```

### Вариант B: Через IDE (IntelliJ IDEA)

1. Открыть проект `hh_aggregate_service` в IntelliJ IDEA
2. Найти класс `HhAggregateServiceApplication`
3. Запустить через Run → Run 'HhAggregateServiceApplication'

### Проверка запуска

```bash
# В новом терминале
curl http://localhost:8080/health
# Должно вернуть: OK
```

---

## Шаг 4: Установка Playwright браузеров (для Go сервиса)

```bash
cd /Users/timofey/Programming/autoresponse_system/hh_autoapply_service

# Установка браузеров Playwright (требуется один раз)
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install

# Установка системных зависимостей (только для Linux)
# go run github.com/playwright-community/playwright-go/cmd/playwright@latest install-deps
```

---

## Шаг 5: Запуск Go сервиса (hh_autoapply_service)

### Вариант A: Запуск через go run

```bash
cd /Users/timofey/Programming/autoresponse_system/hh_autoapply_service

# Запуск
go run cmd/main.go
```

### Вариант B: Сборка и запуск бинарника

```bash
cd /Users/timofey/Programming/autoresponse_system/hh_autoapply_service

# Сборка
go build -o bin/hh_autoapply_service ./cmd/main.go

# Запуск
./bin/hh_autoapply_service
```

### Вариант C: Через IDE (GoLand, VS Code)

1. Открыть проект `hh_autoapply_service`
2. Открыть `cmd/main.go`
3. Запустить через Run → Run

### Проверка запуска

```bash
# В новом терминале
curl http://localhost:8081/health
# Должно вернуть: OK
```

---

## Шаг 6: Проверка работы сервисов

### 6.1. Health Check

```bash
# Java сервис (порт 8080)
curl http://localhost:8080/health

# Go сервис (порт 8081)
curl http://localhost:8081/health
```

Оба должны вернуть: `OK`

---

### 6.2. Регистрация пользователя

```bash
# Через Java сервис
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}'

# Через Go сервис
curl -X POST http://localhost:8081/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}'
```

**Ответ:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiJ9...",
  "expiresAt": "2026-03-13T21:00:00Z"
}
```

**Сохраните токен:**
```bash
export TOKEN="eyJhbGciOiJIUzI1NiJ9..."
```

---

### 6.3. Логин (если пользователь уже существует)

```bash
curl -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}'
```

---

### 6.4. Парсинг вакансий

```bash
# Java сервис
curl -X GET "http://localhost:8080/api/vacancies?query=Java Developer&page=0" \
  -H "Authorization: Bearer $TOKEN"

# Go сервис
curl -X GET "http://localhost:8081/api/vacancies?query=Java Developer&page=0" \
  -H "Authorization: Bearer $TOKEN"
```

**Ответ:**
```json
[
  {
    "id": 12345678,
    "title": "Java Developer",
    "employer": "Яндекс",
    "url": "https://hh.ru/vacancy/12345678",
    "salaryFrom": 150000,
    "salaryTo": 250000,
    "currency": "RUR",
    "region": "Москва"
  }
]
```

---

### 6.5. Извлечение токена HH.ru

⚠️ **Важно:** Для работы с HH.ru нужно сначала авторизоваться на hh.ru в браузере

```bash
# Java сервис
curl -X POST http://localhost:8080/api/hh-token \
  -H "Authorization: Bearer $TOKEN"

# Go сервис
curl -X POST http://localhost:8081/api/hh-token \
  -H "Authorization: Bearer $TOKEN"
```

---

### 6.6. Получение сохранённого токена HH

```bash
curl -X GET http://localhost:8081/api/hh-token \
  -H "Authorization: Bearer $TOKEN"
```

**Ответ:**
```json
{
  "tokenValue": "..."
}
```

---

### 6.7. Создание автоотклика (только Go сервис)

```bash
curl -X POST http://localhost:8081/api/autoapply \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query": "GO Developer", "apply_count": 5}'
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

### 6.8. Проверка статуса автоотклика

```bash
curl -X GET http://localhost:8081/api/autoapply/1 \
  -H "Authorization: Bearer $TOKEN"
```

---

### 6.9. Проверка планировщика

```bash
# Java сервис
curl -X GET http://localhost:8080/api/scheduler/config \
  -H "Authorization: Bearer $TOKEN"

# Go сервис
curl -X GET http://localhost:8081/api/scheduler/config \
  -H "Authorization: Bearer $TOKEN"
```

**Ответ:**
```json
{
  "parserCron": ""
}
```

---

## Шаг 7: Проверка Kafka сообщений

```bash
# Просмотр сообщений в топике vacancies.parsed
docker exec -it hh-autoresponse-kafka /bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic vacancies.parsed \
  --from-beginning
```

---

## Шаг 8: Остановка сервисов

```bash
# Остановка Go сервиса
# Нажмите Ctrl+C в терминале где запущен go run

# Остановка Java сервиса
# Нажмите Ctrl+C в терминале где запущен gradle

# Остановка инфраструктуры (PostgreSQL + Kafka)
cd /Users/timofey/Programming/autoresponse_system/hh_aggregate_service
docker-compose down

# Полная очистка (если нужно)
docker-compose down -v  # Удаляет volumes с данными
```

---

## Сводная таблица портов

| Сервис | Порт | Описание |
|--------|------|----------|
| **hh_aggregate_service (Java)** | 8080 | Основной сервис парсинга |
| **hh_autoapply_service (Go)** | 8081 | Сервис автооткликов |
| **PostgreSQL** | 5444 | База данных |
| **Kafka** | 9092 | Message broker |

---

## API Endpoints

### Аутентификация

| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/api/auth/register` | Регистрация |
| POST | `/api/auth/login` | Логин |

### Вакансии

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/vacancies?query=&page=` | Парсинг вакансий |

### Токен HH

| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/api/hh-token` | Извлечение токена |
| GET | `/api/hh-token` | Получение токена |

### Автоотклик (Go сервис)

| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/api/autoapply` | Создать автоотклик |
| GET | `/api/autoapply/{id}` | Статус автоотклика |

### Планировщик

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/api/scheduler/config` | Конфигурация планировщика |

### Health Check

| Метод | Endpoint | Описание |
|-------|----------|----------|
| GET | `/health` | Проверка здоровья |

---

## Решение проблем

### Ошибка: "connection refused" к PostgreSQL

```bash
# Проверить статус контейнеров
docker-compose ps

# Перезапустить PostgreSQL
docker-compose restart postgres

# Проверить логи
docker-compose logs postgres
```

### Ошибка: "failed to connect to Kafka"

```bash
# Проверить статус Kafka
docker-compose ps kafka

# Перезапустить Kafka
docker-compose restart kafka

# Проверить логи
docker-compose logs kafka
```

### Ошибка: "Playwright browsers not found"

```bash
# Переустановить браузеры
cd /Users/timofey/Programming/autoresponse_system/hh_autoapply_service
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install
```

### Ошибка: "port already in use"

```bash
# Проверить, что использует порт
lsof -i :8080  # Для Java сервиса
lsof -i :8081  # Для Go сервиса

# Остановить процесс
kill -9 <PID>
```

### Ошибка: "TLS handshake timeout" при pull Docker образов

```bash
# Проверить подключение к интернету
# Отключить VPN если включен
# Попробовать еще раз
docker-compose pull
docker-compose up -d
```

---

## Быстрый старт (все команды подряд)

```bash
# 1. Запуск инфраструктуры
cd /Users/timofey/Programming/autoresponse_system/hh_aggregate_service
docker-compose up -d

# 2. Запуск Java сервиса (в фоне)
cd /Users/timofey/Programming/autoresponse_system/hh_aggregate_service
gradle bootRun &

# 3. Установка Playwright
cd /Users/timofey/Programming/autoresponse_system/hh_autoapply_service
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install

# 4. Запуск Go сервиса (в фоне)
cd /Users/timofey/Programming/autoresponse_system/hh_autoapply_service
go run cmd/main.go &

# 5. Проверка
sleep 5
curl http://localhost:8080/health
curl http://localhost:8081/health

# 6. Регистрация и тест
curl -X POST http://localhost:8081/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}'
```

---

## Скрипты для управления

В корне проекта доступны скрипты для управления:

```bash
# Запуск инфраструктуры и подготовка
./start.sh

# Тестирование API (тестирует Go сервис на порту 8081)
./test-api.sh

# Тестирование Java сервиса на порту 8080
./test-api.sh 8080

# Остановка инфраструктуры
./stop.sh
```

---

## Контакты и поддержка

При возникновении проблем:
1. Проверьте логи сервисов
2. Проверьте статус Docker контейнеров
3. Убедитесь, что все порты свободны
4. Проверьте подключение к интернету (для Docker Hub)

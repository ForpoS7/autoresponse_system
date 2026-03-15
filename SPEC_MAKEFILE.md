# Makefile — Specification

## Общая информация

**Расположение:** `/Users/timofey/Programming/autoresponse_system/Makefile`

**Назначение:** Централизованное управление всеми сервисами системы через единую точку входа.

## Доступные команды

### Основные команды

| Команда | Описание |
|---------|----------|
| `make help` | Показать справку по всем доступным командам |
| `make infra` | Запустить инфраструктуру (PostgreSQL + Kafka) |
| `make stop` | Остановить инфраструктуру |
| `make java` | Запустить Java сервис (hh_aggregate_service) |
| `make go` | Запустить Go сервис (hh_autoapply_service) |
| `make build` | Собрать Go сервис |
| `make install-playwright` | Установить браузеры Playwright |
| `make test` | Тестировать API Go сервиса |
| `make test-java` | Тестировать API Java сервиса |
| `make clean` | Очистить временные файлы |
| `make logs` | Показать логи Docker контейнеров |
| `make ps` | Показать статус Docker контейнеров |

## Подробное описание команд

### `make help`

Показывает отформатированную справку по всем доступным командам с использованием цветного вывода.

**Пример вывода:**
```
HH AutoResponse System - Доступные команды

Основные команды:
  help            Показать справку
  infra           Запустить инфраструктуру (PostgreSQL + Kafka)
  stop            Остановить инфраструктуру
  java            Запустить Java сервис (hh_aggregate_service)
  go              Запустить Go сервис (hh_autoapply_service)
  build           Собрать Go сервис
  install-playwright  Установить браузеры Playwright
  test            Тестировать API Go сервиса
  test-java       Тестировать API Java сервиса
  clean           Очистить временные файлы
  logs            Показать логи Docker контейнеров
  ps              Показать статус Docker контейнеров

Пример использования:
  make infra     # Запустить PostgreSQL + Kafka
  make java      # Запустить Java сервис
  make go        # Запустить Go сервис
  make test      # Тестировать API
```

### `make infra`

Запускает Docker контейнеры с PostgreSQL и Kafka.

**Что делает:**
1. Переходит в директорию `hh_aggregate_service`
2. Выполняет `docker-compose up -d`
3. Ожидает 10 секунд для запуска PostgreSQL
4. Показывает статус контейнеров

**Команда:**
```bash
cd hh_aggregate_service && docker-compose up -d
```

**Использование:**
```bash
make infra
```

### `make stop`

Останавливает Docker контейнеры.

**Что делает:**
1. Переходит в директорию `hh_aggregate_service`
2. Выполняет `docker-compose down`

**Команда:**
```bash
cd hh_aggregate_service && docker-compose down
```

**Использование:**
```bash
make stop
```

### `make java`

Запускает Java сервис через Gradle.

**Что делает:**
1. Переходит в директорию `hh_aggregate_service`
2. Выполняет `./gradlew bootRun`

**Команда:**
```bash
cd hh_aggregate_service && ./gradlew bootRun
```

**Использование:**
```bash
make java
```

**Примечание:** Сервис запускается в foreground режиме. Для запуска в фоне используйте `make java &`.

### `make go`

Запускает Go сервис через `go run`.

**Что делает:**
1. Переходит в директорию `hh_autoapply_service`
2. Выполняет `go run cmd/main.go`

**Команда:**
```bash
cd hh_autoapply_service && go run cmd/main.go
```

**Использование:**
```bash
make go
```

**Примечание:** Сервис запускается в foreground режиме. Для запуска в фоне используйте `make go &`.

### `make build`

Собирает Go сервис в бинарный файл.

**Что делает:**
1. Переходит в директорию `hh_autoapply_service`
2. Выполняет `go build -o bin/hh_autoapply_service ./cmd/main.go`

**Команда:**
```bash
cd hh_autoapply_service && go build -o bin/hh_autoapply_service ./cmd/main.go
```

**Результат:**
```
hh_autoapply_service/bin/hh_autoapply_service
```

**Использование:**
```bash
make build
./hh_autoapply_service/bin/hh_autoapply_service
```

### `make install-playwright`

Устанавливает браузеры Playwright для Go сервиса.

**Что делает:**
1. Переходит в директорию `hh_autoapply_service`
2. Выполняет установку браузеров

**Команда:**
```bash
cd hh_autoapply_service && go run github.com/playwright-community/playwright-go/cmd/playwright@latest install
```

**Использование:**
```bash
make install-playwright
```

**Примечание:** Требуется выполнить один раз перед первым запуском Go сервиса.

### `make test`

Тестирует API Go сервиса.

**Что делает:**
1. Выполняет скрипт `./test-api.sh`
2. Скрипт проверяет:
   - Health check
   - Регистрацию пользователя
   - Парсинг вакансий
   - Получение токена HH
   - Конфигурацию планировщика
   - Создание автоотклика

**Команда:**
```bash
./test-api.sh
```

**Использование:**
```bash
make test
```

### `make test-java`

Тестирует API Java сервиса.

**Что делает:**
1. Выполняет скрипт `./test-api.sh 8080`
2. Тестирует Java сервис на порту 8080

**Команда:**
```bash
./test-api.sh 8080
```

**Использование:**
```bash
make test-java
```

### `make clean`

Очищает временные файлы и артефакты сборки.

**Что делает:**
1. Выполняет `go clean` в директории Go сервиса
2. Удаляет директорию `bin/`

**Команда:**
```bash
cd hh_autoapply_service && go clean
rm -rf hh_autoapply_service/bin/
```

**Использование:**
```bash
make clean
```

### `make logs`

Показывает логи Docker контейнеров в реальном времени.

**Что делает:**
1. Переходит в директорию `hh_aggregate_service`
2. Выполняет `docker-compose logs -f`

**Команда:**
```bash
cd hh_aggregate_service && docker-compose logs -f
```

**Использование:**
```bash
make logs
```

### `make ps`

Показывает статус Docker контейнеров.

**Что делает:**
1. Переходит в директорию `hh_aggregate_service`
2. Выполняет `docker-compose ps`

**Команда:**
```bash
cd hh_aggregate_service && docker-compose ps
```

**Использование:**
```bash
make ps
```

## Переменные Makefile

```makefile
# Цвета для вывода
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m
```

## Примеры использования

### Быстрый старт

```bash
# 1. Запуск инфраструктуры
make infra

# 2. Установка браузеров Playwright
make install-playwright

# 3. Запуск Java сервиса (в фоне)
make java &

# 4. Запуск Go сервиса (в фоне)
make go &

# 5. Тестирование
sleep 10
make test
```

### Остановка

```bash
# Остановка инфраструктуры
make stop

# Остановка сервисов (Ctrl+C или kill)
pkill -f "gradle"
pkill -f "go run"
```

### Отладка

```bash
# Проверка статуса контейнеров
make ps

# Просмотр логов
make logs

# Очистка и пересборка
make clean
make build
```

## Структура Makefile

```makefile
.PHONY: help start stop test clean build java go infra

# Цвета для вывода
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m

help: ## Показать справку
    @echo ...

infra: ## Запустить инфраструктуру
    @echo ...
    cd hh_aggregate_service && docker-compose up -d

stop: ## Остановить инфраструктуру
    @echo ...
    cd hh_aggregate_service && docker-compose down

java: ## Запустить Java сервис
    @echo ...
    cd hh_aggregate_service && ./gradlew bootRun

go: ## Запустить Go сервис
    @echo ...
    cd hh_autoapply_service && go run cmd/main.go

build: ## Собрать Go сервис
    @echo ...
    cd hh_autoapply_service && go build ...

install-playwright: ## Установить браузеры Playwright
    @echo ...
    cd hh_autoapply_service && go run ... playwright install

test: ## Тестировать API Go сервиса
    @echo ...
    ./test-api.sh

test-java: ## Тестировать API Java сервиса
    @echo ...
    ./test-api.sh 8080

clean: ## Очистить временные файлы
    @echo ...
    cd hh_autoapply_service && go clean
    rm -rf hh_autoapply_service/bin/

logs: ## Показать логи Docker контейнеров
    @echo ...
    cd hh_aggregate_service && docker-compose logs -f

ps: ## Показать статус Docker контейнеров
    @echo ...
    cd hh_aggregate_service && docker-compose ps
```

## Преимущества использования Makefile

1. **Единая точка входа** — все команды управления в одном месте
2. **Кроссплатформенность** — работает на macOS, Linux, Windows (с Make)
3. **Цветной вывод** — наглядная индикация статуса операций
4. **Автоматизация** — сокращает количество ручных команд
5. **Документация** — встроенная справка через `make help`

---

**Дата создания:** 2026-03-15  
**Версия спецификации:** 1.0

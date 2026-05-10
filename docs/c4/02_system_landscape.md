# Уровень 2 — Ландшафт систем (System Landscape)

> Система в контексте IT-ландшафта. Показывает, какие внешние системы окружают
> HH AutoResponse System, как соискатель взаимодействует со всей экосистемой,
> и что остаётся за границами автоматизации.

```mermaid
flowchart TD
    USER["👤 Соискатель"]

    subgraph AUTOMATED["Автоматизированная зона"]
        direction LR
        SYS["⚙️ HH AutoResponse System
        ────────────────────────────
        Go · Java · Python
        Docker Compose + k3d/k8s"]

        subgraph INFRA["Локальная инфраструктура"]
            direction TB
            KAFKA["Apache Kafka\nKRaft 3.7"]
            PG["PostgreSQL 16"]
            REDIS["Redis 7"]
            MINIO["MinIO (S3)"]
        end

        subgraph OBS["Observability"]
            direction TB
            PROM["Prometheus"]
            GRAF["Grafana"]
        end
    end

    subgraph EXTERNAL["Внешние системы"]
        direction TB
        HH["🌐 HH.ru\nПлатформа поиска работы"]
        OLLAMA["🤖 Ollama\nqwen2.5:7b (локальный LLM)"]
    end

    subgraph OUT_OF_SCOPE["Вне автоматизации"]
        direction TB
        HR["🏢 HR-система\nработодателя"]
        EMAIL["📧 Email-уведомления\nот HH.ru"]
    end

    USER -->|"Настройка критериев\nи запуск"| SYS
    SYS -->|"Отчёт об откликах\nи статусы"| USER

    SYS -->|"Парсинг вакансий\nPlaywright Firefox"| HH
    SYS -->|"Отправка откликов\nPlaywright Firefox"| HH
    SYS -->|"POST /api/generate\n{вакансия + требования}"| OLLAMA
    OLLAMA -->|"Сгенерированное письмо"| SYS

    SYS --- KAFKA
    SYS --- PG
    SYS --- REDIS
    SYS --- MINIO

    PROM -->|"scrape /metrics"| SYS
    GRAF -->|"PromQL"| PROM

    HH -.->|"Решение о приглашении\n(вне контроля системы)"| HR
    HH -.->|"Email соискателю\n(вне контроля системы)"| EMAIL
    EMAIL -.->|"Читает вручную"| USER
```

## Зоны ответственности

| Зона | Что включено | Кто управляет |
|---|---|---|
| Автоматизированная | SYS + Infra + Observability | Команда разработки |
| Внешние системы | HH.ru, Ollama | Третьи стороны |
| Вне автоматизации | HR-системы, Email | HH.ru / Работодатель |

## Ключевые интеграционные точки

| Интеграция | Протокол | Направление | Примечание |
|---|---|---|---|
| HH.ru парсинг | HTTP/Browser (Playwright) | SYS → HH.ru | Нет публичного API |
| HH.ru отклик | HTTP/Browser (Playwright) | SYS → HH.ru | От имени соискателя |
| Ollama генерация | HTTP REST POST /api/generate | SYS → Ollama | Локальный сервер |
| Пользователь | HTTP REST :80 (nginx) | USER ↔ SYS | JWT-аутентификация |

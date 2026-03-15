# HH AutoResponse System — Project Specification

## Обзор проекта

**HH AutoResponse System** — это распределённая система для автоматического парсинга вакансий с HH.ru и автооткликов на них. Система состоит из двух микросервисов, работающих с общей инфраструктурой (PostgreSQL + Kafka).

## Архитектура

```
┌─────────────────────────────────────────────────────────────────┐
│                     HH AutoResponse System                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────┐         ┌──────────────────────┐     │
│  │   hh_aggregate_      │         │   hh_autoapply_      │     │
│  │   service (Java)     │         │   service (Go)       │     │
│  │   Port: 8080         │         │   Port: 8081         │     │
│  │                      │         │                      │     │
│  │  • Парсинг HH.ru     │         │  • Автоотклики       │     │
│  │  • JWT Auth          │         │  • Playwright        │     │
│  │  • Kafka Producer    │         │  • Kafka Consumer    │     │
│  │  • Scheduler         │         │  • AI Cover Letters  │     │
│  └──────────┬───────────┘         └──────────┬───────────┘     │
│             │                                │                  │
│             │         ┌──────────────────────┘                  │
│             │         │                                         │
│             ▼         ▼                                         │
│     ┌──────────────────────────┐                               │
│     │   Apache Kafka           │                               │
│     │   Topics:                │                               │
│     │   - vacancies.parsed     │                               │
│     │   - vacancy_input        │                               │
│     │   - vacancy_output       │                               │
│     │   Port: 9092             │                               │
│     └────────────┬─────────────┘                               │
│                  │                                              │
│     ┌────────────▼─────────────┐     ┌──────────────────────┐  │
│     │   PostgreSQL             │     │   generate_service   │  │
│     │   Port: 5444             │     │   (Python)           │  │
│     │   DB: hh                 │     │   • Ollama LLM       │  │
│     └──────────────────────────┘     │   • Cover Letters    │  │
│                                       └──────────┬───────────┘  │
│                                                  │              │
│                                                  └──────────────┘
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Технологии

| Компонент | Технология | Версия |
|-----------|------------|--------|
| **Java сервис** | Spring Boot 3.2.5 | Java 17+ |
| **Go сервис** | Go | 1.22+ |
| **Python сервис** | Python + Requests | 3.x |
| **LLM** | Ollama (Qwen2.5:7b) | latest |
| **База данных** | PostgreSQL | 16-alpine |
| **Message Broker** | Apache Kafka | 3.7.0 |
| **Browser Automation** | Playwright | 1.58.0 |
| **Build Tool (Java)** | Gradle | 8.x |
| **Router (Go)** | Gorilla Mux | 1.8.1 |

## Инфраструктура

### Docker контейнеры

| Контейнер | Образ | Порт | Описание |
|-----------|-------|------|----------|
| `hh-autoresponse-db` | postgres:16-alpine | 5444:5432 | PostgreSQL |
| `hh-autoresponse-kafka` | apache/kafka:3.7.0 | 9092:9092 | Apache Kafka |

### Kafka Topics

| Топик | Описание | Продюсер | Консьюмер |
|-------|----------|----------|-----------|
| `vacancies.parsed` | Распарсенные вакансии | Java/Go сервис | Go сервис |
| `vacancy_input` | Запросы на генерацию письма | Go сервис | generate_service |
| `vacancy_output` | Сгенерированные письма | generate_service | Go сервис |

---

**Дата создания:** 2026-03-15  
**Версия спецификации:** 1.0

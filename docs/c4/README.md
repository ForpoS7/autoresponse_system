# C4 Architecture Diagrams — HH AutoResponse System

Все диаграммы в формате Mermaid, рендерятся на GitHub без дополнительных инструментов.

| # | Диаграмма | Уровень | Описание |
|---|---|---|---|
| [01](01_context.md) | System Context | C4 L1 | Система + внешние акторы (User, HH.ru, Ollama) |
| [02](02_container.md) | Container | C4 L2 | Все контейнеры: 5 сервисов + 4 хранилища + observability |
| [03](03_component_auth.md) | auth_service | C4 L3 | Handler → Service → Repository + JWTManager |
| [04](04_component_autoapply.md) | hh_autoapply_service | C4 L3 | EDA-оркестрация, кеш, архивация, метрики |
| [05](05_component_aggregate.md) | hh_aggregate_service | C4 L3 | Spring Boot компоненты: REST, Kafka, Playwright |
| [06](06_dynamic_eda_flow.md) | EDA AutoApply Flow | Dynamic (Sequence) | Полный флоу POST /api/autoapply → completed |
| [07](07_dynamic_data_tier.md) | Data Tier Flow | Dynamic (Flowchart) | Hot → Warm → Cold архивация данных |

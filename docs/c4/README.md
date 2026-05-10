# C4 Architecture Diagrams — HH AutoResponse System

Все диаграммы в формате Mermaid, рендерятся на GitHub без дополнительных инструментов.
Иерархия из 7 уровней: от бизнес-контекста до классов/кода.

| # | Файл | Уровень | Что показывает |
|---|---|---|---|
| [01](01_business_context.md) | Business Context | L1 | Система как «чёрный ящик» — акторы, внешние системы, бизнес-процессы |
| [02](02_system_landscape.md) | System Landscape | L2 | IT-ландшафт — все системы и интеграции в экосистеме соискателя |
| [03](03_containers.md) | Containers | L3 | Исполняемые единицы: 5 сервисов, 4 хранилища, observability, протоколы |
| [04](04_subcontainers.md) | Subcontainers | L4 | Внутренние Go-модули autoapply_service: cmd/, internal/, pkg/ |
| [05](05_components.md) | Components | L5 | Группы классов/функций внутри каждого сервиса |
| [06](06_modules_packages.md) | Modules / Packages | L6 | Структура исходного кода: директории и файлы всех сервисов |
| [07](07_classes_code.md) | Classes / Code | L7 | UML classDiagram: Kafka-сообщения, Correlation ID, AutoApply домен |

## Навигация по уровням

```
L1 Business Context  →  "Зачем нужна система и кто ей пользуется?"
L2 System Landscape  →  "Какие системы рядом и как они связаны?"
L3 Containers        →  "Из каких процессов состоит система?"
L4 Subcontainers     →  "Как устроен изнутри самый сложный контейнер?"
L5 Components        →  "Какие группы логики есть в каждом сервисе?"
L6 Modules/Packages  →  "Как организован исходный код по папкам?"
L7 Classes/Code      →  "Как выглядят ключевые структуры данных?"
```

## Технологии диаграмм

| Тип | Используется для |
|---|---|
| `flowchart LR` | Горизонтальные потоки (Business Context, System Landscape) |
| `flowchart TD` | Иерархические слои (Containers, Modules/Packages) |
| `classDiagram` | UML-структуры данных (Classes/Code) |

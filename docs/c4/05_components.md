# Уровень 5 — Компоненты (Components)

> Группы классов и функций внутри каждого сервиса.
> Диаграммы разбиты по сервисам для читаемости.

## 5.1 auth_service (Go)

```mermaid
flowchart LR
    subgraph AUTH_SVC["auth_service · Go 1.22 · :8082"]
        direction TB
        R["router\ngorilla/mux\nPOST /register\nPOST /login\nPOST /validate\nGET  /health"]
        H["AuthHandler\nDecodeJSON → validate\n→ Service → JWT"]
        S["AuthService\nHashPassword (bcrypt)\nCheckPassword\nGenerateToken (HS256)\nValidateToken"]
        REPO["UserRepository\nGetByUsername()\nCreate(user)"]
    end

    CLIENT["nginx / другие сервисы"]
    PG_U["PostgreSQL\ntable: users"]

    CLIENT -->|"HTTP JSON"| R
    R --> H
    H --> S
    S --> REPO
    REPO -->|"lib/pq\nSELECT/INSERT"| PG_U
    S -->|"JWT HS256\nBearer token"| H
```

## 5.2 hh_aggregate_service (Java / Spring Boot)

```mermaid
flowchart LR
    subgraph AGGR_SVC["hh_aggregate_service · Java 21 · :8080"]
        direction TB
        FILTER["JwtAuthFilter\n(Spring Security)\nValidateToken → SecurityContext"]
        CTRL_V["VacancyController\nGET /api/vacancies\nparams: query, page"]
        CTRL_T["TokenController\nGET  /api/hh-token\nPOST /api/hh-token"]
        CTRL_S["SchedulerController\nPOST /api/scheduler/start\nGET  /api/scheduler/status"]

        PW_SVC["PlaywrightService\nFirefox · headless\nLogin(storageState)\nParsePage(query,page)\n→ List<Vacancy>"]
        TOKEN_SVC["HhTokenService\nExtractToken(username,pass)\n→ storageState JSON"]
        SCHED["SchedulerService\n@Scheduled (закомментирован)\nRunJob(userId, query)"]

        PARSE_CONS["ParseRequestConsumer\nKafka ← parse.requested\n→ ParsePage → publish vacancies.parsed"]
        TOKEN_PUB["TokenPublisher\nKafka → token.updated\n{userId, storageState}"]
        VAC_PUB["VacanciesBatchPublisher\nKafka → vacancies.parsed\n{jobId, List<Vacancy>}"]

        REPO_T["HhTokenRepository\nJPA · hh_tokens table"]
        REPO_V["VacancyRepository\nJPA · vacancies table"]
    end

    NGINX["nginx :8080"]
    KAFKA_A["Kafka\n→ vacancies.parsed\n→ token.updated\n← parse.requested"]
    PG_A["PostgreSQL\nhh_tokens · vacancies"]
    HH_A["HH.ru"]

    NGINX --> FILTER
    FILTER --> CTRL_V
    FILTER --> CTRL_T
    FILTER --> CTRL_S

    CTRL_V --> PW_SVC
    CTRL_T --> TOKEN_SVC
    CTRL_S --> SCHED

    PARSE_CONS --> PW_SVC
    PARSE_CONS --> VAC_PUB
    TOKEN_SVC --> TOKEN_PUB

    PW_SVC -->|"Browser"| HH_A
    TOKEN_SVC --> REPO_T
    CTRL_V --> REPO_V

    REPO_T -->|"JPA"| PG_A
    REPO_V -->|"JPA"| PG_A

    PARSE_CONS -->|"← parse.requested"| KAFKA_A
    VAC_PUB    -->|"→ vacancies.parsed"| KAFKA_A
    TOKEN_PUB  -->|"→ token.updated"| KAFKA_A
```

## 5.3 generate_service (Python)

```mermaid
flowchart LR
    subgraph GEN_SVC["generate_service · Python 3.11"]
        direction TB
        CONS["KafkaConsumer\nkafka-python\n← vacancy_input\ngroup: generate-group"]
        PROC["LetterProcessor\nПарсинг JSON\nформирование prompt"]
        OLLAMA_CL["OllamaClient\nPOST /api/generate\nmodel: qwen2.5:7b\nstream: false"]
        PROD["KafkaProducer\nkafka-python\n→ vacancy_output\n{correlationId, letter}"]
    end

    KAFKA_G["Kafka\n← vacancy_input\n→ vacancy_output"]
    OLLAMA_G["Ollama :11434\nqwen2.5:7b"]

    KAFKA_G -->|"JSON\n{corrId,title,company,requirements}"| CONS
    CONS --> PROC
    PROC --> OLLAMA_CL
    OLLAMA_CL -->|"POST /api/generate"| OLLAMA_G
    OLLAMA_G -->|"generated text"| OLLAMA_CL
    OLLAMA_CL --> PROD
    PROD -->|"JSON\n{corrId, letter}"| KAFKA_G
```

## 5.4 Observability Stack

```mermaid
flowchart LR
    subgraph OBSERVABILITY["Observability"]
        direction TB
        PROM_CFG["prometheus.yml\nscrape_configs:\n- job: autoapply :8081/metrics\n- job: prometheus :9090/metrics"]
        PROM_DB["Prometheus TSDB\n/prometheus/data\nметрики за 15 дней"]
        GRAF_DS["Grafana Datasource\nPrometheus (auto-provisioned)"]
        GRAF_DASH["Grafana Dashboards\nAutoResponse Platform\nKubernetes Overview"]
    end

    APPLY_M["hh_autoapply_service\n/metrics (promhttp)"]
    USER_G["Инженер / DevOps"]

    APPLY_M -->|"Prometheus exposition\nformat"| PROM_CFG
    PROM_CFG --> PROM_DB
    PROM_DB --> GRAF_DS
    GRAF_DS --> GRAF_DASH
    USER_G -->|"Browser :3000"| GRAF_DASH
    USER_G -->|"Browser :9090"| PROM_DB
```

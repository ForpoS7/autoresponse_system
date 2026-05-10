# Уровень 6 — Модули / Пакеты (Modules / Packages)

> Структура исходного кода: директории, файлы и их назначение.

## 6.1 hh_autoapply_service (Go)

```mermaid
flowchart TD
    ROOT_GO["hh_autoapply_service/"]

    subgraph CMD["cmd/"]
        MAIN["main.go\nточка входа\nDI + HTTP server"]
    end

    subgraph INTERNAL["internal/"]
        direction LR
        subgraph INT_H["handler/"]
            HH1["autoapply_handler.go"]
            HH2["token_handler.go"]
            HH3["archiver_handler.go"]
        end
        subgraph INT_S["service/"]
            SS1["autoapply_service.go"]
            SS2["archiver_service.go"]
        end
        subgraph INT_MW["middleware/"]
            MW1["metrics.go"]
        end
        subgraph INT_C["cache/"]
            CC1["token_cache.go"]
        end
    end

    subgraph PKG["pkg/"]
        direction LR
        subgraph PK_K["kafka/"]
            KK1["topics.go"]
            KK2["messages.go"]
            KK3["parse_publisher.go"]
            KK4["vacancy_correlator.go"]
            KK5["letter_publisher.go"]
            KK6["letter_correlator.go"]
            KK7["token_consumer.go"]
        end
        subgraph PK_R["redis/"]
            RR1["client.go"]
        end
        subgraph PK_PW["playwright/"]
            PW1["autoapply.go"]
        end
        subgraph PK_M["minio/"]
            MC1["client.go"]
        end
        subgraph PK_MT["metrics/"]
            MT1["metrics.go"]
        end
    end

    ROOT_GO --> CMD
    ROOT_GO --> INTERNAL
    ROOT_GO --> PKG

    MAIN --> INT_H
    MAIN --> INT_S
    MAIN --> PKG

    INT_H --> INT_S
    INT_H --> PK_R
    INT_S --> INT_C
    INT_S --> PK_K
    INT_S --> PK_PW
    INT_C --> PK_R
    INT_S --> PK_M
```

## 6.2 hh_aggregate_service (Java / Maven)

```mermaid
flowchart TD
    ROOT_JAVA["hh_aggregate_service/\nsrc/main/java/\ncom/example/hh/"]

    subgraph CONFIG["config/"]
        J1["SecurityConfig.java"]
        J2["KafkaConsumerConfig.java"]
        J3["KafkaProducerConfig.java"]
    end

    subgraph CONTROLLER["controller/"]
        J4["VacancyController.java"]
        J5["TokenController.java"]
        J6["SchedulerController.java"]
    end

    subgraph SERVICE["service/"]
        J7["PlaywrightService.java"]
        J8["HhTokenService.java"]
        J9["SchedulerService.java"]
    end

    subgraph KAFKA_J["kafka/"]
        J10["ParseRequestConsumer.java"]
        J11["TokenPublisher.java"]
        J12["VacanciesBatchPublisher.java"]
    end

    subgraph MODEL["model/"]
        J13["Vacancy.java"]
        J14["HhToken.java"]
        J15["ParseRequestMessage.java"]
        J16["TokenUpdatedMessage.java"]
        J17["VacanciesBatch.java"]
    end

    subgraph REPO["repository/"]
        J18["VacancyRepository.java"]
        J19["HhTokenRepository.java"]
    end

    subgraph SECURITY["security/"]
        J20["JwtAuthFilter.java"]
        J21["JwtService.java"]
    end

    ROOT_JAVA --> CONFIG
    ROOT_JAVA --> CONTROLLER
    ROOT_JAVA --> SERVICE
    ROOT_JAVA --> KAFKA_J
    ROOT_JAVA --> MODEL
    ROOT_JAVA --> REPO
    ROOT_JAVA --> SECURITY

    CONTROLLER --> SERVICE
    KAFKA_J --> SERVICE
    SERVICE --> REPO
    KAFKA_J --> MODEL
    CONTROLLER --> MODEL
```

## 6.3 auth_service (Go)

```mermaid
flowchart TD
    ROOT_AUTH["auth_service/"]

    subgraph AUTH_CMD["cmd/"]
        A1["main.go"]
    end
    subgraph AUTH_H["handler/"]
        A2["auth_handler.go\nRegister/Login/Validate"]
    end
    subgraph AUTH_S["service/"]
        A3["auth_service.go\nBCrypt + JWT"]
    end
    subgraph AUTH_R["repository/"]
        A4["user_repository.go\nPostgreSQL lib/pq"]
    end
    subgraph AUTH_M["model/"]
        A5["user.go"]
        A6["token.go"]
    end

    ROOT_AUTH --> AUTH_CMD
    ROOT_AUTH --> AUTH_H
    ROOT_AUTH --> AUTH_S
    ROOT_AUTH --> AUTH_R
    ROOT_AUTH --> AUTH_M

    AUTH_H --> AUTH_S
    AUTH_S --> AUTH_R
```

## 6.4 Инфраструктура проекта

```mermaid
flowchart TD
    ROOT["autoresponse_system/"]

    subgraph DC["docker-compose.yml\nnginx · auth · postgres\nkafka · redis · minio\nprometheus · grafana"]
    end

    subgraph NGINX_DIR["nginx/"]
        N1["nginx.conf\nrate limiting\nupstream blocks"]
    end

    subgraph PROM_DIR["prometheus/"]
        P1["prometheus.yml\nscrape_configs"]
    end

    subgraph GRAF_DIR["grafana/"]
        G1["provisioning/\ndatasources/prometheus.yml"]
    end

    subgraph K8S["k8s/"]
        direction LR
        subgraph K8S_CH["charts/\nautoapply · auth\naggregate · generate\nHelm values.yaml"]
        end
        subgraph K8S_OB["observability/\nvictoriametrics\nloki · tempo · grafana"]
        end
        subgraph K8S_CI["cicd/\nsetup.ps1\nKaniko RBAC\nARC runners"]
        end
    end

    subgraph TESTS["tests/"]
        direction LR
        T1["load/\nlocustfile.py\nrun.ps1"]
        T2["chaos/\nfault-injection.yaml\ncircuit-breaker-test.ps1"]
    end

    ROOT --> DC
    ROOT --> NGINX_DIR
    ROOT --> PROM_DIR
    ROOT --> GRAF_DIR
    ROOT --> K8S
    ROOT --> TESTS
```

# C4 Dynamic — EDA AutoApply Flow

Полный флоу запроса автоотклика от POST до статуса `completed`.  
Показывает как контейнеры взаимодействуют через Kafka во времени.

```mermaid
sequenceDiagram
  autonumber
  actor User as Соискатель
  participant nginx as nginx<br/>(API Gateway)
  participant Go as hh_autoapply_service<br/>:8081
  participant Redis as Redis<br/>:6379
  participant Kafka as Apache Kafka<br/>:9092
  participant Java as hh_aggregate_service<br/>:8080
  participant Python as generate_service
  participant HH as HH.ru
  participant Ollama as Ollama<br/>(qwen2.5:7b)

  User->>nginx: POST /api/autoapply<br/>{query, userId, applyCount}
  nginx->>Go: proxy (rate limit: 60 rpm)
  Go->>Go: DB: INSERT auto_apply_requests<br/>status = "pending"
  Go-->>User: 202 {requestId, status: "pending"}

  Note over Go: горутина processAutoApply(requestId)

  Go->>Redis: GET hh_token:{userId}
  alt Cache hit (24h TTL)
    Redis-->>Go: storageState (Playwright session)
  else Cache miss
    Go->>Go: DB: HhTokenRepository.GetByUserID
  end

  Go->>Kafka: PUBLISH parse.requested<br/>{jobId, query, page=1, userId}
  Kafka->>Java: CONSUME parse.requested
  Java->>HH: Playwright Firefox:<br/>searchVacancies(query, area=1)
  HH-->>Java: HTML → []Vacancy

  Java->>Kafka: PUBLISH vacancies.parsed<br/>{jobId, userId, []Vacancy}
  Java->>Kafka: PUBLISH token.updated<br/>{userId, storageState}

  Kafka->>Go: CONSUME vacancies.parsed<br/>(VacancyCorrelator routes by jobId)
  Kafka->>Go: CONSUME token.updated<br/>(TokenConsumer → TokenCache.Set)

  Go->>Go: DB: UPDATE status = "in_progress"

  loop для каждой вакансии (до applyCount)
    Go->>Kafka: PUBLISH vacancy_input<br/>{correlationId, title, company, requirements}
    Kafka->>Python: CONSUME vacancy_input
    Python->>Ollama: POST /api/generate<br/>{model: qwen2.5:7b, prompt}
    Ollama-->>Python: generated_text
    Python->>Kafka: PUBLISH vacancy_output<br/>{correlationId, generated_text}
    Kafka->>Go: CONSUME vacancy_output<br/>(LetterCorrelator routes by correlationId)

    alt Письмо получено (≤ 30s)
      Go->>HH: Playwright Firefox:<br/>applyToVacancy(storageState, letter)
      HH-->>Go: 200 OK (отклик отправлен)
    else Timeout generate_service
      Go->>HH: Playwright Firefox:<br/>applyToVacancy(storageState, fallbackLetter)
    end

    Go->>Go: DB: INSERT auto_apply_logs
  end

  Go->>Go: DB: UPDATE status = "completed"<br/>applied_count = N
  Go->>Redis: SET autoapply:{id} TTL=∞<br/>(terminal status, кешируем навсегда)

  User->>nginx: GET /api/autoapply/{id}
  nginx->>Go: proxy
  Go->>Redis: GET autoapply:{id}
  Redis-->>Go: {status: "completed", appliedCount: N}
  Go-->>User: 200 {status: "completed", appliedCount: N}
```

## Таймауты

| Операция | Таймаут | Что происходит при превышении |
|---|---|---|
| WaitForJob (vacancies.parsed) | 2 мин | `processAutoApply` завершается с ошибкой |
| WaitForLetter (vacancy_output) | 30 сек | Используется fallback-письмо из захардкоженного списка |
| Playwright applyToVacancy | ~15 сек | Ошибка логируется, вакансия пропускается |

## Kafka correlation pattern

```
parse.requested  → jobId как key  → VacancyCorrelator.pending sync.Map[jobId → chan]
vacancy_input    → corrId как key → LetterCorrelator.pending  sync.Map[corrId → chan]
```
Каждый канал ждёт ровно одно сообщение, после чего закрывается. Race-free.

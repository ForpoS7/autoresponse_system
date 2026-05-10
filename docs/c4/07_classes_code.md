# Уровень 7 — Классы / Код (Classes / Code)

> UML-диаграммы ключевых структур данных и их отношений.
> Диаграммы разбиты по доменам для читаемости.

## 7.1 EDA — Kafka сообщения (Go)

```mermaid
classDiagram
    class ParseRequestMessage {
        +string JobID
        +string Query
        +int    Page
        +string UserID
    }

    class VacanciesBatch {
        +string      JobID
        +[]Vacancy   Vacancies
        +int         Total
    }

    class Vacancy {
        +int    ID
        +string Title
        +string Company
        +string Requirements
        +string URL
        +int    Salary
    }

    class LetterRequest {
        +string CorrelationID
        +string Title
        +string Company
        +string Requirements
        +string UserResume
    }

    class LetterResponse {
        +string CorrelationID
        +string Letter
        +string Error
    }

    class TokenUpdatedMessage {
        +string UserID
        +string StorageState
        +int64  UpdatedAt
    }

    VacanciesBatch "1" --> "*" Vacancy : contains
```

## 7.2 Correlation ID паттерн (Go sync.Map)

```mermaid
classDiagram
    class VacancyCorrelator {
        -sync.Map pending
        +Register(jobID string) chan[]Vacancy
        +WaitForJob(jobID string, timeout) []Vacancy
        -consumeLoop()
        -dispatch(batch VacanciesBatch)
    }

    class LetterCorrelator {
        -sync.Map pending
        +Register(corrID string) chan string
        +WaitForLetter(corrID string, timeout) string
        -consumeLoop()
        -dispatch(resp LetterResponse)
    }

    class ParsePublisher {
        -kafka.Writer writer
        +Publish(msg ParseRequestMessage) error
    }

    class LetterPublisher {
        -kafka.Writer writer
        +Publish(msg LetterRequest) error
    }

    class TokenConsumer {
        -kafka.Reader reader
        -TokenCache   cache
        +Start(ctx context.Context)
        -handleMessage(msg TokenUpdatedMessage)
    }

    class TokenCache {
        -sync.Map    memory
        -RedisClient redis
        +Set(userID string, state string)
        +Get(userID string) string, error
        -tryRedis(userID string) string, error
    }

    TokenConsumer --> TokenCache : updates
    VacancyCorrelator --> VacanciesBatch : receives
    LetterCorrelator --> LetterResponse : receives
```

## 7.3 AutoApply домен (Go)

```mermaid
classDiagram
    class AutoApplyRequest {
        +int    ID
        +string UserID
        +string Query
        +int    Page
        +string Status
        +time.Time CreatedAt
        +time.Time UpdatedAt
    }

    class AutoApplyLog {
        +int    ID
        +int    RequestID
        +int    VacancyID
        +string Status
        +string CoverLetter
        +time.Time AppliedAt
    }

    class AutoApplyHandler {
        -AutoApplyService service
        -RedisClient      redis
        +CreateAutoApply(w, r)
        +GetAutoApplyStatus(w, r)
    }

    class AutoApplyService {
        -TokenCache        tokenCache
        -ParsePublisher    parsePub
        -VacancyCorrelator vacCorr
        -LetterPublisher   letterPub
        -LetterCorrelator  letterCorr
        -PlaywrightService playwright
        -AutoApplyRepo     repo
        +CreateAutoApplyRequest(req) AutoApplyRequest, error
        +GetAutoApplyRequest(id) AutoApplyRequest, error
        -runFlow(ctx, req)
        -applyToVacancy(token, vacancy, letter)
    }

    class AutoApplyRepository {
        -db *sql.DB
        +Create(req) error
        +GetByID(id) AutoApplyRequest, error
        +UpdateStatus(id, status) error
        +GetOldLogs(days) []AutoApplyLog, error
        +DeleteLogsByIDs(ids) error
    }

    AutoApplyHandler --> AutoApplyService : calls
    AutoApplyService --> AutoApplyRepository : reads/writes
    AutoApplyRequest "1" --> "*" AutoApplyLog : has
```

## 7.4 Archiver (Go)

```mermaid
classDiagram
    class ArchiverService {
        -AutoApplyRepository repo
        -VacancyRepository   vacRepo
        -MinioClient         minio
        +ArchiveLogs() error
        +ArchiveVacancies() error
        +RunAll() ArchiveResult
        +StartScheduler(interval time.Duration)
    }

    class MinioClient {
        -minio.Client client
        -string       bucket
        +Upload(key string, data []byte) error
        +List(prefix string) []ObjectInfo, error
        -ensureBucket() error
    }

    class ArchiveResult {
        +int ArchivedLogs
        +int ArchivedVacancies
        +[]string Errors
        +time.Time CompletedAt
    }

    class ArchiverHandler {
        -ArchiverService service
        +RunAll(w, r)
        +ListCold(w, r)
    }

    ArchiverHandler --> ArchiverService : calls
    ArchiverService --> MinioClient : uploads
    ArchiverService --> ArchiveResult : returns
```

## 7.5 Kafka сообщения (Java)

```mermaid
classDiagram
    class ParseRequestMessage {
        +String jobId
        +String query
        +int    page
        +String userId
    }

    class VacanciesBatch {
        +String         jobId
        +List~Vacancy~  vacancies
        +int            total
    }

    class TokenUpdatedMessage {
        +String userId
        +String storageState
        +long   updatedAt
    }

    class ParseRequestConsumer {
        -PlaywrightService playwright
        -VacanciesBatchPublisher publisher
        +consume(ParseRequestMessage msg)
    }

    class TokenPublisher {
        -KafkaTemplate template
        +publish(userId, storageState)
    }

    class VacanciesBatchPublisher {
        -KafkaTemplate template
        +publish(VacanciesBatch batch)
    }

    ParseRequestConsumer --> PlaywrightService : uses
    ParseRequestConsumer --> VacanciesBatchPublisher : publishes via
    VacanciesBatch "1" --> "*" Vacancy : contains
```

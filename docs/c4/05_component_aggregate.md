# C4 Level 3 — Component: hh_aggregate_service

Внутренние компоненты Java/Spring Boot сервиса агрегации вакансий.

```mermaid
C4Component
  title Component Diagram — hh_aggregate_service (Java 21 · Spring Boot 3.2.5) :8080

  ContainerDb(postgres, "PostgreSQL", "postgres:16")
  ContainerDb(kafka, "Apache Kafka", "KRaft 3.7")
  System_Ext(hh_ru, "HH.ru")
  Container(go_svc, "hh_autoapply_service", "Go :8081")

  Container_Boundary(svc, "hh_aggregate_service") {

    Component(jwt_filter, "JwtAuthFilter", "Spring Security", "Валидирует JWT на каждый входящий запрос.\nHS256 secret совпадает с auth_service и Go-сервисом")

    Component(vacancy_ctrl, "VacancyController", "Spring REST", "GET /api/vacancies?query=&page=\n→ инициирует парсинг и возвращает вакансии")
    Component(token_ctrl, "TokenController", "Spring REST", "GET  /api/hh-token → получить актуальный токен\nPOST /api/hh-token → запустить Playwright для извлечения")
    Component(scheduler_ctrl, "SchedulerController", "Spring REST", "POST /api/scheduler/start /stop\n(планировщик закомментирован)")

    Component(parser_svc, "ParserService", "Spring @Service", "parseVacancies(query, page, userId)\nparseVacancies(query, page, userId, jobId) ← из Kafka\ngetDescription(url) — детали вакансии")
    Component(token_svc, "TokenService", "Spring @Service", "getToken(userId): DB lookup\nextractToken(userId): Playwright логин + сохранение сессии\nAfter extract → TokenPublisher.publish()")
    Component(vacancy_svc, "VacancyService", "Spring @Service", "saveVacancy, findByQuery\nОбёртка над VacancyRepository")

    Component(vacancy_pub, "VacancyPublisher", "Spring Kafka · KafkaTemplate", "→ vacancies.parsed\nПубликует VacanciesBatch{jobId, userId, List<Vacancy>}")
    Component(token_pub, "TokenPublisher", "Spring Kafka · KafkaTemplate", "→ token.updated\nПубликует TokenUpdatedMessage{userId, storageState}")
    Component(parse_cons, "ParseRequestConsumer", "Spring Kafka · @KafkaListener", "← parse.requested\nПолучает ParseRequestMessage{jobId, query, page, userId}\n→ парсит → VacancyPublisher.publish(VacanciesBatch)")

    Component(playwright_svc, "PlaywrightService", "playwright-java · Firefox", "login(userId): загружает storageState\nsearchVacancies(query, area): парсит список\ngetVacancyDescription(url): парсит детали")

    Component(repos, "Repositories", "Spring Data JPA", "VacancyRepository: save, findByQuery\nHhTokenRepository: findByUserId, save storageState")
  }

  Rel(jwt_filter, vacancy_ctrl, "authenticate")
  Rel(jwt_filter, token_ctrl, "authenticate")

  Rel(vacancy_ctrl, parser_svc, "parseVacancies(query, page, userId)")
  Rel(token_ctrl, token_svc, "getToken / extractToken")
  Rel(scheduler_ctrl, parser_svc, "scheduled parse (disabled)")

  Rel(parser_svc, playwright_svc, "searchVacancies, getDescription")
  Rel(parser_svc, vacancy_pub, "publish(VacanciesBatch)")
  Rel(parser_svc, repos, "saveVacancy")

  Rel(token_svc, playwright_svc, "login + extractToken")
  Rel(token_svc, repos, "getByUserId, save storageState")
  Rel(token_svc, token_pub, "publish(userId, storageState)")

  Rel(parse_cons, kafka, "← parse.requested")
  Rel(parse_cons, parser_svc, "parseVacancies(…, jobId)")

  Rel(vacancy_pub, kafka, "→ vacancies.parsed")
  Rel(token_pub, kafka, "→ token.updated")

  Rel(repos, postgres, "JPA / Hibernate")
  Rel(playwright_svc, hh_ru, "Playwright Firefox browser")
```

## Kafka-топики (продюсер)

| Топик | Сообщение | Когда |
|---|---|---|
| `vacancies.parsed` | `VacanciesBatch{jobId, userId, List<Vacancy>}` | После каждого парсинга |
| `token.updated` | `TokenUpdatedMessage{userId, storageState}` | После `extractToken` |

## Kafka-топики (консьюмер)

| Топик | Триггер |
|---|---|
| `parse.requested` | Go публикует при `POST /api/autoapply` → Java парсит и возвращает |

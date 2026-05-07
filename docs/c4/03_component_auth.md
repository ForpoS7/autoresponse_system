# C4 Level 3 — Component: auth_service

Внутренние компоненты Go сервиса аутентификации.

```mermaid
C4Component
  title Component Diagram — auth_service (Go 1.22) :8082

  Container(nginx, "nginx", "API Gateway", "Проксирует /auth/* с rate limit 5 rpm")
  ContainerDb(postgres, "PostgreSQL", "postgres:16", "table: users (id, email, password_hash, created_at)")

  Container_Boundary(auth, "auth_service") {

    Component(auth_handler, "AuthHandler", "Go · gorilla/mux", "POST /register → Register\nPOST /login    → Login\nPOST /validate → Validate\nGET  /health")

    Component(auth_service, "AuthService", "Go", "Register: проверяет email, BCrypt-хеширует пароль, сохраняет, возвращает JWT\nLogin: сравнивает BCrypt, возвращает JWT\nValidate: парсит и проверяет JWT")

    Component(user_repo, "UserRepository", "Go · lib/pq", "Create(email, passwordHash)\nGetByEmail(email)\nExistsByEmail(email)")

    Component(jwt_mgr, "JWTManager", "golang-jwt/v5", "Generate(userID, email) → signed HS256 token\nValidate(token) → Claims{UserID, Email, ExpiresAt}")
  }

  Rel(nginx, auth_handler, "proxy_pass :8082 (strip /auth/ prefix)")
  Rel(auth_handler, auth_service, "вызывает Register / Login / Validate")
  Rel(auth_service, user_repo, "GetByEmail, ExistsByEmail, Create")
  Rel(auth_service, jwt_mgr, "Generate token после регистрации/входа\nValidate token в /validate")
  Rel(user_repo, postgres, "SQL: INSERT, SELECT", "lib/pq")
```

## Контракт API

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| POST | `/auth/register` | `{email, password}` | `{token, expires_at}` |
| POST | `/auth/login` | `{email, password}` | `{token, expires_at}` |
| POST | `/auth/validate` | `{token}` | `{valid, user_id, email}` |
| GET | `/auth/health` | — | `{status: "ok"}` |

## Замечания

- JWT secret (`HS256`) совпадает с Java и Go сервисами — принимают токены друг друга
- Дублирует аутентификацию в Java и Go (known issue — не трогаем)

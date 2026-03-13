package model

// DTOs для аутентификации
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"` // ISO 8601 формат как в Java сервисе
}

// DTO для парсинга вакансий
type ParserRequest struct {
	Query string `json:"query"`
	Page  int    `json:"page"`
}

// DTO для автоотклика
type AutoApplyRequestDTO struct {
	UserID     int64  `json:"user_id"`
	Query      string `json:"query"`
	ApplyCount int    `json:"apply_count"`
}

// DTO для ответа автоотклика
type AutoApplyResponse struct {
	RequestID    int64  `json:"request_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	AppliedCount int    `json:"applied_count,omitempty"`
	FailedCount  int    `json:"failed_count,omitempty"`
}

// DTO для токена HH
type TokenResponse struct {
	TokenValue string `json:"tokenValue"` // camelCase как в Java сервисе
}

// DTO для scheduler
type SchedulerConfigResponse struct {
	ParserCron string `json:"parserCron"`
}

package model

type User struct {
	ID           int64  `db:"id"`
	Email        string `db:"email"`
	PasswordHash string `db:"password_hash"`
}

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
	Email     string `json:"email"`
	ExpiresAt int64  `json:"expires_at"`
}

// ValidateRequest — внутренний endpoint для других сервисов
type ValidateRequest struct {
	Token string `json:"token"`
}

type ValidateResponse struct {
	Valid  bool   `json:"valid"`
	UserID int64  `json:"user_id,omitempty"`
	Email  string `json:"email,omitempty"`
}

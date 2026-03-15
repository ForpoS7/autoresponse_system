package model

import "time"

type AutoApplyRequest struct {
	ID           int64     `json:"id" db:"id"`
	UserID       int64     `json:"user_id" db:"user_id"`
	Query        string    `json:"query" db:"query"`
	ApplyCount   int       `json:"apply_count" db:"apply_count"`
	AppliedCount int       `json:"applied_count" db:"applied_count"`
	Status       string    `json:"status" db:"status"` // pending, processing, completed, failed
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type AutoApplyLog struct {
	ID           int64     `json:"id" db:"id"`
	RequestID    int64     `json:"request_id" db:"request_id"`
	VacancyID    int64     `json:"vacancy_id" db:"vacancy_id"`
	VacancyURL   string    `json:"vacancy_url" db:"vacancy_url"`
	CoverLetter  string    `json:"cover_letter" db:"cover_letter"`
	Status       string    `json:"status" db:"status"` // success, failed
	ErrorMessage string    `json:"error_message,omitempty" db:"error_message"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

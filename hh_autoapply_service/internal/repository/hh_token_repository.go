package repository

import (
	"database/sql"
	"fmt"
	"time"

	"hh_autoapply_service/internal/model"
)

type HhTokenRepository struct {
	db *sql.DB
}

func NewHhTokenRepository(db *sql.DB) *HhTokenRepository {
	return &HhTokenRepository{db: db}
}

func (r *HhTokenRepository) Save(token *model.HhToken) error {
	// Не сохраняем пустые токены
	if token.TokenValue == "" || len(token.TokenValue) < 100 {
		return fmt.Errorf("token value is too short (%d chars)", len(token.TokenValue))
	}

	query := `
		INSERT INTO hh_tokens (user_id, token_value, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			token_value = EXCLUDED.token_value,
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`
	now := time.Now()
	token.CreatedAt = now
	token.UpdatedAt = now
	return r.db.QueryRow(query, token.UserID, token.TokenValue, now, now).Scan(&token.ID)
}

func (r *HhTokenRepository) GetByUserID(userID int64) (*model.HhToken, error) {
	query := `SELECT id, user_id, token_value, created_at, updated_at FROM hh_tokens WHERE user_id = $1`
	token := &model.HhToken{}
	err := r.db.QueryRow(query, userID).Scan(
		&token.ID, &token.UserID, &token.TokenValue, &token.CreatedAt, &token.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return token, nil
}

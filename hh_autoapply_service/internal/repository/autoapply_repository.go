package repository

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
	"hh_autoapply_service/internal/model"
)

type AutoApplyRepository struct {
	db *sql.DB
}

func NewAutoApplyRepository(db *sql.DB) *AutoApplyRepository {
	return &AutoApplyRepository{db: db}
}

func (r *AutoApplyRepository) CreateRequest(req *model.AutoApplyRequest) error {
	query := `
		INSERT INTO auto_apply_requests (user_id, query, apply_count, applied_count, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	now := time.Now()
	req.CreatedAt = now
	req.UpdatedAt = now
	return r.db.QueryRow(
		query,
		req.UserID, req.Query, req.ApplyCount, req.AppliedCount, req.Status, now, now,
	).Scan(&req.ID)
}

func (r *AutoApplyRepository) UpdateRequest(req *model.AutoApplyRequest) error {
	query := `
		UPDATE auto_apply_requests
		SET applied_count = $1, status = $2, updated_at = $3
		WHERE id = $4
	`
	req.UpdatedAt = time.Now()
	_, err := r.db.Exec(query, req.AppliedCount, req.Status, req.UpdatedAt, req.ID)
	return err
}

func (r *AutoApplyRepository) GetRequestByID(id int64) (*model.AutoApplyRequest, error) {
	query := `SELECT id, user_id, query, apply_count, applied_count, status, created_at, updated_at FROM auto_apply_requests WHERE id = $1`
	req := &model.AutoApplyRequest{}
	err := r.db.QueryRow(query, id).Scan(
		&req.ID, &req.UserID, &req.Query, &req.ApplyCount, &req.AppliedCount,
		&req.Status, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (r *AutoApplyRepository) CreateLog(log *model.AutoApplyLog) error {
	query := `
		INSERT INTO auto_apply_logs (request_id, vacancy_id, vacancy_url, cover_letter, status, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	now := time.Now()
	return r.db.QueryRow(
		query,
		log.RequestID, log.VacancyID, log.VacancyURL, log.CoverLetter,
		log.Status, log.ErrorMessage, now,
	).Scan(&log.ID)
}

// GetOldLogs возвращает логи старше olderThanDays дней (warm → cold).
func (r *AutoApplyRepository) GetOldLogs(olderThanDays int) ([]model.AutoApplyLog, error) {
	query := `
		SELECT id, request_id, vacancy_id, vacancy_url, cover_letter, status, error_message, created_at
		FROM auto_apply_logs
		WHERE created_at < NOW() - ($1 || ' days')::interval
		ORDER BY created_at
	`
	rows, err := r.db.Query(query, olderThanDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.AutoApplyLog
	for rows.Next() {
		var l model.AutoApplyLog
		if err := rows.Scan(&l.ID, &l.RequestID, &l.VacancyID, &l.VacancyURL,
			&l.CoverLetter, &l.Status, &l.ErrorMessage, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// DeleteLogsByIDs удаляет логи после успешной архивации в cold storage.
func (r *AutoApplyRepository) DeleteLogsByIDs(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	query := `DELETE FROM auto_apply_logs WHERE id = ANY($1)`
	_, err := r.db.Exec(query, pq.Array(ids))
	return err
}

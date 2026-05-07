package repository

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
	"hh_autoapply_service/internal/model"
)

type VacancyRepository struct {
	db *sql.DB
}

func NewVacancyRepository(db *sql.DB) *VacancyRepository {
	return &VacancyRepository{db: db}
}

func (r *VacancyRepository) Create(vacancy *model.Vacancy) error {
	query := `
		INSERT INTO vacancies (title, employer, url, description, salary_from, salary_to, currency, region, user_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	now := time.Now()
	return r.db.QueryRow(
		query,
		vacancy.Title, vacancy.Employer, vacancy.URL, vacancy.Description,
		vacancy.SalaryFrom, vacancy.SalaryTo, vacancy.Currency, vacancy.Region,
		vacancy.UserID, now,
	).Scan(&vacancy.ID)
}

func (r *VacancyRepository) CreateMany(vacancies []model.Vacancy) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO vacancies (title, employer, url, description, salary_from, salary_to, currency, region, user_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	for _, v := range vacancies {
		now := time.Now()
		_, err := tx.Exec(
			query,
			v.Title, v.Employer, v.URL, v.Description,
			v.SalaryFrom, v.SalaryTo, v.Currency, v.Region,
			v.UserID, now,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetOldVacancies возвращает вакансии старше olderThanDays дней.
func (r *VacancyRepository) GetOldVacancies(olderThanDays int) ([]model.Vacancy, error) {
	query := `
		SELECT id, title, employer, url, description,
		       salary_from, salary_to, currency, region, user_id
		FROM vacancies
		WHERE created_at < NOW() - ($1 || ' days')::interval
		ORDER BY created_at
	`
	rows, err := r.db.Query(query, olderThanDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vacancies []model.Vacancy
	for rows.Next() {
		var v model.Vacancy
		if err := rows.Scan(&v.ID, &v.Title, &v.Employer, &v.URL, &v.Description,
			&v.SalaryFrom, &v.SalaryTo, &v.Currency, &v.Region, &v.UserID); err != nil {
			return nil, err
		}
		vacancies = append(vacancies, v)
	}
	return vacancies, rows.Err()
}

// DeleteVacanciesByIDs удаляет вакансии после архивации в cold storage.
func (r *VacancyRepository) DeleteVacanciesByIDs(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query := `DELETE FROM vacancies WHERE id::text = ANY($1)`
	_, err := r.db.Exec(query, pq.Array(ids))
	return err
}

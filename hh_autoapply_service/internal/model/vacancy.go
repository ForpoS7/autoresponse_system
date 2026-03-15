package model

type Vacancy struct {
	ID          int64  `json:"id" db:"id"`
	Title       string `json:"title" db:"title"`
	Employer    string `json:"employer" db:"employer"`
	URL         string `json:"url" db:"url"`
	Description string `json:"description,omitempty" db:"description"`
	SalaryFrom  *int64 `json:"salaryFrom,omitempty" db:"salary_from"`
	SalaryTo    *int64 `json:"salaryTo,omitempty" db:"salary_to"`
	Currency    string `json:"currency,omitempty" db:"currency"`
	Region      string `json:"region,omitempty" db:"region"`
	UserID      int64  `json:"user_id" db:"user_id"`
}

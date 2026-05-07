package kafka

// VacanciesBatch — формат сообщения в топике vacancies.parsed.
// jobId связывает запрос parse.requested с результатом.
type VacanciesBatch struct {
	JobID     string            `json:"jobId"`
	UserID    int64             `json:"userId"`
	Vacancies []VacancyPayload  `json:"vacancies"`
}

type VacancyPayload struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Employer    string  `json:"employer"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	SalaryFrom  *int64  `json:"salaryFrom,omitempty"`
	SalaryTo    *int64  `json:"salaryTo,omitempty"`
	Currency    string  `json:"currency,omitempty"`
	Region      string  `json:"region,omitempty"`
	UserID      int64   `json:"userId"`
}

// ParseRequestMessage — Go публикует в parse.requested, Java исполняет.
type ParseRequestMessage struct {
	JobID  string `json:"jobId"`
	Query  string `json:"query"`
	Page   int    `json:"page"`
	UserID int64  `json:"userId"`
}

// TokenUpdatedMessage — Java публикует в token.updated при сохранении сессии.
type TokenUpdatedMessage struct {
	UserID     int64  `json:"userId"`
	TokenValue string `json:"tokenValue"`
}

// LetterRequest — Go публикует в vacancy_input для generate_service.
type LetterRequest struct {
	CorrelationID string `json:"correlationId"`
	Title         string `json:"title"`
	Company       string `json:"company"`
	Requirements  string `json:"requirements"`
}

// LetterResponse — generate_service публикует в vacancy_output.
type LetterResponse struct {
	CorrelationID string `json:"correlationId"`
	GeneratedText string `json:"generated_text"`
}

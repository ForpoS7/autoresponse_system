package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type HHAggregateClient struct {
	baseURL    string
	httpClient *http.Client
}

type HHTokenResponse struct {
	TokenValue string `json:"tokenValue"`
}

// Java сервис возвращает массив вакансий напрямую
type VacancyItem struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Employer    string `json:"employer"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	SalaryFrom  *int64 `json:"salaryFrom,omitempty"`
	SalaryTo    *int64 `json:"salaryTo,omitempty"`
	Currency    string `json:"currency,omitempty"`
	Region      string `json:"region,omitempty"`
	UserID      int64  `json:"userId"`
}

func NewHHAggregateClient(baseURL string, timeout time.Duration) *HHAggregateClient {
	// Увеличиваем таймаут до 60 секунд
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &HHAggregateClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *HHAggregateClient) GetHHToken(ctx context.Context, userID int64, jwtToken string) (*HHTokenResponse, error) {
	url := fmt.Sprintf("%s/api/hh-token", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// JWT токен обязателен - из него Java сервис получает user_id
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp HHTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}

func (c *HHAggregateClient) GetVacancies(ctx context.Context, userID int64, jwtToken string, query string, page int) ([]VacancyItem, error) {
	// Кодируем query параметр для URL
	encodedQuery := url.QueryEscape(query)
	urlStr := fmt.Sprintf("%s/api/vacancies?query=%s&page=%d", c.baseURL, encodedQuery, page)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// JWT токен обязателен - из него Java сервис получает user_id
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Java сервис возвращает массив вакансий напрямую
	var vacancies []VacancyItem
	if err := json.NewDecoder(resp.Body).Decode(&vacancies); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return vacancies, nil
}

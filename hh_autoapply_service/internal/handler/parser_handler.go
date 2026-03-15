package handler

import (
	"encoding/json"
	"hh_autoapply_service/internal/middleware"
	"hh_autoapply_service/internal/service"
	"net/http"
	"strconv"
)

type ParserHandler struct {
	parserService *service.ParserService
}

func NewParserHandler(parserService *service.ParserService) *ParserHandler {
	return &ParserHandler{parserService: parserService}
}

// GetVacancies godoc
// @Summary Parse vacancies from HH.ru
// @Tags vacancies
// @Accept json
// @Produce json
// @Param query query string false "Search query" default(Java Developer)
// @Param page query int false "Page number" default(0)
// @Success 200 {array} model.Vacancy
// @Security BearerAuth
// @Router /api/vacancies [get]
func (h *ParserHandler) GetVacancies(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r)
	if userID == 0 {
		http.Error(w, `{"error": "user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		query = "Java Developer"
	}

	pageStr := r.URL.Query().Get("page")
	page := 0
	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			http.Error(w, `{"error": "invalid page parameter"}`, http.StatusBadRequest)
			return
		}
	}

	vacancies, err := h.parserService.ParseVacancies(r.Context(), query, page, userID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vacancies)
}

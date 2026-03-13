package handler

import (
	"encoding/json"
	"hh_autoapply_service/internal/middleware"
	"hh_autoapply_service/internal/model"
	"hh_autoapply_service/internal/service"
	"net/http"
)

type TokenHandler struct {
	tokenService *service.TokenService
}

func NewTokenHandler(tokenService *service.TokenService) *TokenHandler {
	return &TokenHandler{tokenService: tokenService}
}

// GetHHToken godoc
// @Summary Get HH.ru token
// @Tags token
// @Produce json
// @Success 200 {object} model.TokenResponse
// @Security BearerAuth
// @Router /api/hh-token [get]
func (h *TokenHandler) GetHHToken(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r)
	if userID == 0 {
		http.Error(w, `{"error": "user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	token, err := h.tokenService.GetToken(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error": "token not found"}`, http.StatusNotFound)
		return
	}

	response := model.TokenResponse{
		TokenValue: token.TokenValue,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ExtractHHToken godoc
// @Summary Extract HH.ru token from browser session
// @Tags token
// @Produce json
// @Success 200 {object} model.TokenResponse
// @Security BearerAuth
// @Router /api/hh-token [post]
func (h *TokenHandler) ExtractHHToken(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r)
	if userID == 0 {
		http.Error(w, `{"error": "user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	tokenValue, err := h.tokenService.ExtractToken(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response := model.TokenResponse{
		TokenValue: tokenValue,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

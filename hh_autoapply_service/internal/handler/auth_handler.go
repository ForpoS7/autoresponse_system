package handler

import (
	"encoding/json"
	"hh_autoapply_service/internal/model"
	"hh_autoapply_service/internal/service"
	"net/http"
	"time"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.RegisterRequest true "Register request"
// @Success 200 {object} model.AuthResponse
// @Failure 400 {object} map[string]string
// @Router /api/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error": "email and password are required"}`, http.StatusBadRequest)
		return
	}

	token, expiresAt, err := h.authService.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			http.Error(w, `{"error": "user already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error": "failed to register user"}`, http.StatusInternalServerError)
		return
	}

	// Конвертируем timestamp в ISO 8601 формат
	expiresAtTime := time.UnixMilli(expiresAt)
	response := model.AuthResponse{
		Token:     token,
		ExpiresAt: expiresAtTime.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Login godoc
// @Summary Login user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.LoginRequest true "Login request"
// @Success 200 {object} model.AuthResponse
// @Failure 400 {object} map[string]string
// @Router /api/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error": "email and password are required"}`, http.StatusBadRequest)
		return
	}

	token, expiresAt, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			http.Error(w, `{"error": "invalid email or password"}`, http.StatusUnauthorized)
			return
		}
		http.Error(w, `{"error": "failed to login"}`, http.StatusInternalServerError)
		return
	}

	// Конвертируем timestamp в ISO 8601 формат
	expiresAtTime := time.UnixMilli(expiresAt)
	response := model.AuthResponse{
		Token:     token,
		ExpiresAt: expiresAtTime.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

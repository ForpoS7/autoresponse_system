package handler

import (
	"encoding/json"
	"hh_autoapply_service/internal/config"
	"hh_autoapply_service/internal/middleware"
	"hh_autoapply_service/internal/model"
	"net/http"
)

type SchedulerHandler struct {
	config *config.SchedulerConfig
}

func NewSchedulerHandler(cfg *config.SchedulerConfig) *SchedulerHandler {
	return &SchedulerHandler{config: cfg}
}

// GetSchedulerConfig godoc
// @Summary Get scheduler configuration
// @Tags scheduler
// @Produce json
// @Success 200 {object} model.SchedulerConfigResponse
// @Security BearerAuth
// @Router /api/scheduler/config [get]
func (h *SchedulerHandler) GetSchedulerConfig(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r)
	if userID == 0 {
		http.Error(w, `{"error": "user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	response := model.SchedulerConfigResponse{
		ParserCron: h.config.Parser.Cron,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

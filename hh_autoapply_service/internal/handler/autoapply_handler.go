package handler

import (
	"encoding/json"
	"hh_autoapply_service/internal/model"
	"hh_autoapply_service/internal/service"
	"net/http"
	"strconv"
)

type AutoApplyHandler struct {
	autoApplyService *service.AutoApplyService
}

func NewAutoApplyHandler(autoApplyService *service.AutoApplyService) *AutoApplyHandler {
	return &AutoApplyHandler{autoApplyService: autoApplyService}
}

func (h *AutoApplyHandler) CreateAutoApply(w http.ResponseWriter, r *http.Request) {
	var req model.AutoApplyRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, `{"error": "query is required"}`, http.StatusBadRequest)
		return
	}

	if req.ApplyCount <= 0 {
		http.Error(w, `{"error": "apply_count must be positive"}`, http.StatusBadRequest)
		return
	}

	userID := req.UserID
	if userID == 0 {
		http.Error(w, `{"error": "user_id is required"}`, http.StatusBadRequest)
		return
	}

	autoApplyReq, err := h.autoApplyService.CreateAutoApplyRequest(r.Context(), userID, req.Query, req.ApplyCount)
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	response := model.AutoApplyResponse{
		RequestID:    autoApplyReq.ID,
		Status:       autoApplyReq.Status,
		Message:      "Auto-apply process started",
		AppliedCount: autoApplyReq.AppliedCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AutoApplyHandler) GetAutoApplyStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	requestID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid request id"}`, http.StatusBadRequest)
		return
	}

	req, err := h.autoApplyService.GetAutoApplyRequest(r.Context(), requestID)
	if err != nil {
		http.Error(w, `{"error": "request not found"}`, http.StatusNotFound)
		return
	}

	response := model.AutoApplyResponse{
		RequestID:    req.ID,
		Status:       req.Status,
		Message:      "",
		AppliedCount: req.AppliedCount,
		FailedCount:  req.ApplyCount - req.AppliedCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

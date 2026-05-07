package handler

import (
	"encoding/json"
	"fmt"
	"hh_autoapply_service/internal/model"
	"hh_autoapply_service/internal/service"
	"hh_autoapply_service/pkg/metrics"
	"hh_autoapply_service/pkg/redis"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

const autoApplyStatusTTL = time.Hour

type AutoApplyHandler struct {
	autoApplyService *service.AutoApplyService
	cache            *redis.Client // nil when Redis is unavailable
}

func NewAutoApplyHandler(autoApplyService *service.AutoApplyService, cache *redis.Client) *AutoApplyHandler {
	return &AutoApplyHandler{autoApplyService: autoApplyService, cache: cache}
}

func (h *AutoApplyHandler) CreateAutoApply(w http.ResponseWriter, r *http.Request) {
	var req model.AutoApplyRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query is required"})
		return
	}
	if req.ApplyCount <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "apply_count must be positive"})
		return
	}
	if req.UserID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id is required"})
		return
	}

	autoApplyReq, err := h.autoApplyService.CreateAutoApplyRequest(r.Context(), req.UserID, req.Query, req.ApplyCount)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	metrics.AutoApplyRequestsTotal.WithLabelValues("created").Inc()

	response := model.AutoApplyResponse{
		RequestID:    autoApplyReq.ID,
		Status:       autoApplyReq.Status,
		Message:      "Auto-apply process started",
		AppliedCount: autoApplyReq.AppliedCount,
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AutoApplyHandler) GetAutoApplyStatus(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	requestID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request id"})
		return
	}

	cacheKey := fmt.Sprintf("autoapply:%d", requestID)

	// 1. Cache hit
	if h.cache != nil {
		var cached model.AutoApplyResponse
		if err := h.cache.GetJSON(r.Context(), cacheKey, &cached); err == nil {
			metrics.RedisCacheHits.WithLabelValues("autoapply").Inc()
			writeJSON(w, http.StatusOK, cached)
			return
		}
		metrics.RedisCacheMisses.WithLabelValues("autoapply").Inc()
	}

	// 2. DB lookup
	req, err := h.autoApplyService.GetAutoApplyRequest(r.Context(), requestID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "request not found"})
		return
	}

	response := model.AutoApplyResponse{
		RequestID:    req.ID,
		Status:       req.Status,
		Message:      "",
		AppliedCount: req.AppliedCount,
		FailedCount:  req.ApplyCount - req.AppliedCount,
	}

	// 3. Cache write — только для завершённых статусов кешируем навсегда (TTL=0 — нет изменений)
	if h.cache != nil {
		ttl := autoApplyStatusTTL
		if req.Status == "completed" || req.Status == "failed" {
			ttl = 0 // no expiry for terminal statuses
		}
		if err := h.cache.SetJSON(r.Context(), cacheKey, response, ttl); err != nil {
			// non-fatal: log and continue
			_ = err
		}
	}

	writeJSON(w, http.StatusOK, response)
}

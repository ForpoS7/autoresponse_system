package handler

import (
	"encoding/json"
	"hh_autoapply_service/internal/service"
	"net/http"
)

type ArchiverHandler struct {
	svc *service.ArchiverService
}

func NewArchiverHandler(svc *service.ArchiverService) *ArchiverHandler {
	return &ArchiverHandler{svc: svc}
}

// RunArchive godoc
// POST /api/archive/run — запускает полный цикл архивации вручную.
func (h *ArchiverHandler) RunArchive(w http.ResponseWriter, r *http.Request) {
	result := h.svc.RunAll(r.Context())
	status := http.StatusOK
	if result.Error != "" {
		status = http.StatusInternalServerError
	}
	writeJSON(w, status, result)
}

// ListArchive godoc
// GET /api/archive/list?prefix=logs  — список файлов в cold storage.
func (h *ArchiverHandler) ListArchive(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	objects, err := h.svc.ListCold(r.Context(), prefix)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"prefix":  prefix,
		"count":   len(objects),
		"objects": objects,
	})
}

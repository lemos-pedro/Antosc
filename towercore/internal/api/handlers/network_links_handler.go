package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"towercore/internal/core/interfaces"
)

// NetworkLinksHandler orquestra apenas request/response, sem regra de
// negócio, seguindo a regra "handlers HTTP sem lógica de negócio" definida
// em contributing.md.
type NetworkLinksHandler struct {
	linkRepo  interfaces.NetworkLinkRepository
	eventRepo interfaces.NetworkLinkEventRepository
}

func NewNetworkLinksHandler(linkRepo interfaces.NetworkLinkRepository, eventRepo interfaces.NetworkLinkEventRepository) *NetworkLinksHandler {
	return &NetworkLinksHandler{linkRepo: linkRepo, eventRepo: eventRepo}
}

// ListLinks trata GET /api/v1/links
func (h *NetworkLinksHandler) ListLinks(w http.ResponseWriter, r *http.Request) {
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntOrDefault(r.URL.Query().Get("offset"), 0)

	links, total, err := h.linkRepo.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "falha ao listar links")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": links,
		"meta": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
}

// GetLink trata GET /api/v1/links/{link_id}
func (h *NetworkLinksHandler) GetLink(w http.ResponseWriter, r *http.Request) {
	linkID := r.PathValue("link_id")

	link, err := h.linkRepo.GetByID(r.Context(), linkID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "falha ao obter link")
		return
	}
	if link == nil {
		writeError(w, http.StatusNotFound, "RESOURCE_NOT_FOUND", "link não encontrado")
		return
	}

	writeJSON(w, http.StatusOK, link)
}

// GetLinkEvents trata GET /api/v1/links/{link_id}/events
func (h *NetworkLinksHandler) GetLinkEvents(w http.ResponseWriter, r *http.Request) {
	linkID := r.PathValue("link_id")
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntOrDefault(r.URL.Query().Get("offset"), 0)

	events, total, err := h.eventRepo.ListByLinkID(r.Context(), linkID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "falha ao listar eventos do link")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": events,
		"meta": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
}

func parseIntOrDefault(raw string, def int) int {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"towercore/internal/core/services"
)

type ZabbixLinksHandler struct {
	service       *services.ZabbixLinkSyncService
	defaultSearch []string
}

func NewZabbixLinksHandler(service *services.ZabbixLinkSyncService, defaultSearch []string) *ZabbixLinksHandler {
	return &ZabbixLinksHandler{service: service, defaultSearch: defaultSearch}
}

type zabbixLinkRequest struct {
	HostSearch []string `json:"host_search"`
}

func (h *ZabbixLinksHandler) Sync(w http.ResponseWriter, r *http.Request) {
	searches := h.searches(r)
	result, err := h.service.Sync(r.Context(), searches)
	if err != nil {
		writeError(w, http.StatusBadGateway, "ZABBIX_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *ZabbixLinksHandler) Inspect(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Inspect(r.Context(), h.searches(r))
	if err != nil {
		writeError(w, http.StatusBadGateway, "ZABBIX_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *ZabbixLinksHandler) searches(r *http.Request) []string {
	searches := append([]string(nil), h.defaultSearch...)
	if raw := r.URL.Query().Get("host_search"); raw != "" {
		searches = splitSearch(raw)
	}
	if r.Method == http.MethodPost && r.Body != nil {
		var req zabbixLinkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && len(req.HostSearch) > 0 {
			searches = req.HostSearch
		}
	}
	return searches
}
func splitSearch(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

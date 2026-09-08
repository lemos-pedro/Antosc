package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"towercore/internal/core/domain"
	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type BackhaulInterfaceHandler struct {
	service *services.BackhaulInterfaceService
}

func NewBackhaulInterfaceHandler(service *services.BackhaulInterfaceService) *BackhaulInterfaceHandler {
	return &BackhaulInterfaceHandler{service: service}
}

func (h *BackhaulInterfaceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.collect(w, r)
	case http.MethodGet:
		h.list(w, r)
	default:
		apierror.MethodNotAllowed(w)
	}
}

func (h *BackhaulInterfaceHandler) collect(w http.ResponseWriter, r *http.Request) {
	var payload json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}
	var batch []*domain.BackhaulInterface
	if err := json.Unmarshal(payload, &batch); err == nil {
		if err := h.service.CollectMultipleInterfaces(r.Context(), batch); err != nil {
			apierror.BadRequest(w, err.Error())
			return
		}
		writeHandlerJSON(w, http.StatusCreated, map[string]any{"status": "ok", "count": len(batch)})
		return
	}
	var iface domain.BackhaulInterface
	if err := json.Unmarshal(payload, &iface); err != nil {
		apierror.BadRequest(w, "expected a backhaul object or array")
		return
	}
	if err := h.service.CollectInterface(r.Context(), &iface); err != nil {
		apierror.BadRequest(w, err.Error())
		return
	}
	writeHandlerJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

func (h *BackhaulInterfaceHandler) list(w http.ResponseWriter, r *http.Request) {
	towerID, ok := queryUUID(w, r, "tower_id")
	if !ok {
		return
	}
	filter := &domain.BackhaulInterfaceFilter{TowerID: towerID, Limit: queryInt(r, "limit", 100), Offset: queryInt(r, "offset", 0), OrderBy: []string{"measured_at DESC"}}
	if name := r.URL.Query().Get("interface_name"); name != "" {
		filter.InterfaceName = &name
	}
	if from, ok := queryTime(w, r, "from"); !ok {
		return
	} else {
		filter.MeasuredAtAfter = from
	}
	if to, ok := queryTime(w, r, "to"); !ok {
		return
	} else {
		filter.MeasuredAtBefore = to
	}
	items, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		apierror.Internal(w)
		return
	}
	writeHandlerJSON(w, http.StatusOK, map[string]any{"data": items, "count": len(items), "total": total, "limit": filter.Limit, "offset": filter.Offset})
}

func (h *BackhaulInterfaceHandler) GetTowerBackhaulStatus(w http.ResponseWriter, r *http.Request) {
	towerID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		apierror.BadRequest(w, "invalid tower id")
		return
	}
	items, err := h.service.GetTowerBackhaulStatus(r.Context(), towerID)
	if err != nil {
		apierror.Internal(w)
		return
	}
	writeHandlerJSON(w, http.StatusOK, map[string]any{"data": items, "count": len(items)})
}

func queryUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	v, err := uuid.Parse(r.URL.Query().Get(name))
	if err != nil {
		apierror.BadRequest(w, "invalid "+name)
		return uuid.Nil, false
	}
	return v, true
}
func queryInt(r *http.Request, name string, fallback int) int {
	if v, err := strconv.Atoi(r.URL.Query().Get(name)); err == nil && v >= 0 {
		return v
	}
	return fallback
}
func queryTime(w http.ResponseWriter, r *http.Request, name string) (*time.Time, bool) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		apierror.BadRequest(w, "invalid "+name)
		return nil, false
	}
	return &t, true
}

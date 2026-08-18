package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"towercore/internal/core/domain"
	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type SiteEnvironmentHandler struct{ service *services.SiteEnvironmentService }

func NewSiteEnvironmentHandler(service *services.SiteEnvironmentService) *SiteEnvironmentHandler {
	return &SiteEnvironmentHandler{service: service}
}

func (h *SiteEnvironmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.collect(w, r)
	case http.MethodGet:
		h.list(w, r)
	default:
		apierror.MethodNotAllowed(w)
	}
}

func (h *SiteEnvironmentHandler) collect(w http.ResponseWriter, r *http.Request) {
	var payload json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil { apierror.BadRequest(w, "invalid json payload"); return }
	var batch []*domain.SiteEnvironment
	if err := json.Unmarshal(payload, &batch); err == nil {
		if err := h.service.CollectMultipleEnvironment(r.Context(), batch); err != nil { apierror.BadRequest(w, err.Error()); return }
		writeJSON(w, http.StatusCreated, map[string]any{"status": "ok", "count": len(batch)})
		return
	}
	var env domain.SiteEnvironment
	if err := json.Unmarshal(payload, &env); err != nil { apierror.BadRequest(w, "expected a site environment object or array"); return }
	if err := h.service.CollectEnvironment(r.Context(), &env); err != nil { apierror.BadRequest(w, err.Error()); return }
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

func (h *SiteEnvironmentHandler) list(w http.ResponseWriter, r *http.Request) {
	siteID, ok := queryUUID(w, r, "site_id"); if !ok { return }
	filter := &domain.SiteEnvironmentFilter{SiteID: siteID, Limit: queryInt(r, "limit", 100), Offset: queryInt(r, "offset", 0), OrderBy: []string{"measured_at DESC"}}
	if from, ok := queryTime(w, r, "from"); !ok { return } else { filter.MeasuredAtAfter = from }
	if to, ok := queryTime(w, r, "to"); !ok { return } else { filter.MeasuredAtBefore = to }
	setEnvironmentBooleanFilters(r, filter)
	items, total, err := h.service.GetEnvironmentHistory(r.Context(), siteID, filter.Limit, filter.Offset, filter.MeasuredAtAfter, filter.MeasuredAtBefore)
	if err != nil { apierror.Internal(w); return }
	items = filterEnvironment(items, filter)
	writeJSON(w, http.StatusOK, map[string]any{"data": items, "count": len(items), "total": total, "limit": filter.Limit, "offset": filter.Offset})
}

func (h *SiteEnvironmentHandler) GetSiteEnvironmentStatus(w http.ResponseWriter, r *http.Request) {
	siteID, err := uuid.Parse(r.PathValue("id")); if err != nil { apierror.BadRequest(w, "invalid site id"); return }
	env, err := h.service.GetLatestEnvironment(r.Context(), siteID); if err != nil { apierror.Internal(w); return }
	writeJSON(w, http.StatusOK, map[string]any{"data": env})
}

func setEnvironmentBooleanFilters(r *http.Request, f *domain.SiteEnvironmentFilter) {
	if r.URL.Query().Get("high_temp") == "true" { n := 35.0; f.InternalTempCMin = &n }
	if r.URL.Query().Get("high_humidity") == "true" { n := 80.0; f.HumidityPctMin = &n }
	if r.URL.Query().Get("door_open") == "true" { b := true; f.DoorOpen = &b }
	if r.URL.Query().Get("mains_down") == "true" { b := false; f.MainsPowerOk = &b }
	if r.URL.Query().Get("ups_on_battery") == "true" { b := true; f.UpsOnBattery = &b }
	if r.URL.Query().Get("smoke_detected") == "true" { b := true; f.SmokeDetected = &b }
}

func filterEnvironment(items []*domain.SiteEnvironment, f *domain.SiteEnvironmentFilter) []*domain.SiteEnvironment {
	result := make([]*domain.SiteEnvironment, 0, len(items))
	for _, item := range items {
		if f.InternalTempCMin != nil && (item.InternalTempC == nil || *item.InternalTempC < *f.InternalTempCMin) { continue }
		if f.HumidityPctMin != nil && (item.HumidityPct == nil || *item.HumidityPct < *f.HumidityPctMin) { continue }
		if f.DoorOpen != nil && (item.DoorOpen == nil || *item.DoorOpen != *f.DoorOpen) { continue }
		if f.MainsPowerOk != nil && (item.MainsPowerOk == nil || *item.MainsPowerOk != *f.MainsPowerOk) { continue }
		if f.UpsOnBattery != nil && (item.UpsOnBattery == nil || *item.UpsOnBattery != *f.UpsOnBattery) { continue }
		if f.SmokeDetected != nil && (item.SmokeDetected == nil || *item.SmokeDetected != *f.SmokeDetected) { continue }
		result = append(result, item)
	}
	return result
}

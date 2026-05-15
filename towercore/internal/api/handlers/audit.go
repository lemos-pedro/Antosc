package handlers

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"database/sql"
	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type AuditHandler struct {
	service *services.AuditService
}

func NewAuditHandler(service *services.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

func (h *AuditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apierror.MethodNotAllowed(w)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/export.csv") {
		h.exportCSV(w, r)
		return
	}

	if id := strings.TrimSpace(r.PathValue("id")); id != "" {
		h.getByID(w, r, id)
		return
	}

	h.list(w, r)
}

func (h *AuditHandler) list(w http.ResponseWriter, r *http.Request) {

	filter := interfaces.AuditFilter{
		Actor:    strings.TrimSpace(r.URL.Query().Get("actor")),
		Action:   strings.TrimSpace(r.URL.Query().Get("action")),
		Resource: strings.TrimSpace(r.URL.Query().Get("resource")),
		Limit:    parseIntDefault(r.URL.Query().Get("limit"), 50),
		Offset:   parseIntDefault(r.URL.Query().Get("offset"), 0),
	}

	if rawFrom := strings.TrimSpace(r.URL.Query().Get("from")); rawFrom != "" {
		from, err := time.Parse(time.RFC3339, rawFrom)
		if err != nil {
			apierror.BadRequest(w, "from must be RFC3339")
			return
		}
		filter.From = &from
	}
	if rawTo := strings.TrimSpace(r.URL.Query().Get("to")); rawTo != "" {
		to, err := time.Parse(time.RFC3339, rawTo)
		if err != nil {
			apierror.BadRequest(w, "to must be RFC3339")
			return
		}
		filter.To = &to
	}

	logs, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		apierror.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(logs)
}

func (h *AuditHandler) getByID(w http.ResponseWriter, r *http.Request, id string) {
	entry, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			apierror.Write(w, http.StatusNotFound, "not_found", "audit log not found")
			return
		}
		apierror.BadRequest(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(entry)
}

func (h *AuditHandler) exportCSV(w http.ResponseWriter, r *http.Request) {
	filter := interfaces.AuditFilter{
		Actor:    strings.TrimSpace(r.URL.Query().Get("actor")),
		Action:   strings.TrimSpace(r.URL.Query().Get("action")),
		Resource: strings.TrimSpace(r.URL.Query().Get("resource")),
		Limit:    parseIntDefault(r.URL.Query().Get("limit"), 5000),
		Offset:   parseIntDefault(r.URL.Query().Get("offset"), 0),
	}

	if rawFrom := strings.TrimSpace(r.URL.Query().Get("from")); rawFrom != "" {
		from, err := time.Parse(time.RFC3339, rawFrom)
		if err != nil {
			apierror.BadRequest(w, "from must be RFC3339")
			return
		}
		filter.From = &from
	}
	if rawTo := strings.TrimSpace(r.URL.Query().Get("to")); rawTo != "" {
		to, err := time.Parse(time.RFC3339, rawTo)
		if err != nil {
			apierror.BadRequest(w, "to must be RFC3339")
			return
		}
		filter.To = &to
	}

	logs, _, err := h.service.List(r.Context(), filter)
	if err != nil {
		apierror.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="audit-logs.csv"`)
	w.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"audit_id", "actor", "action", "resource", "resource_id", "details", "created_at"})
	for _, l := range logs {
		_ = writer.Write([]string{
			l.ID,
			l.Actor,
			l.Action,
			l.Resource,
			l.ResourceID,
			l.Details,
			l.CreatedAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}

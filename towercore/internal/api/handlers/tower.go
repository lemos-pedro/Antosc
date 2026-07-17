package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"towercore/internal/api/middleware"
	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type TowerHandler struct {
	service      *services.TowerService
	availability *services.AvailabilityService
}

func NewTowerHandler(service *services.TowerService, availability *services.AvailabilityService) *TowerHandler {
	return &TowerHandler{service: service, availability: availability}
}

func (h *TowerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if strings.TrimSpace(r.PathValue("id")) != "" {
			h.getByID(w, r)
			return
		}
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	case http.MethodPatch:
		h.configureSNMP(w, r)
	default:
		apierror.MethodNotAllowed(w)
	}
}

type createTowerRequest struct {
	ID              string  `json:"tower_id"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	OperatorID      string  `json:"operator_id"`
	RegionID        string  `json:"region_id"`
	Vendor          string  `json:"vendor"`
	SNMPEnabled     bool    `json:"snmp_enabled"`
	SNMPVersion     string  `json:"snmp_version"`
	SNMPTarget      string  `json:"snmp_target"`
	SNMPCommunity   string  `json:"snmp_community"`
	SNMPV3User      string  `json:"snmp_v3_user"`
	SNMPAuthProto   string  `json:"snmp_auth_protocol"`
	SNMPAuthPass    string  `json:"snmp_auth_password"`
	SNMPPrivProto   string  `json:"snmp_priv_protocol"`
	SNMPPrivPass    string  `json:"snmp_priv_password"`
	Availability30d float64 `json:"availability_30d"`
}

type configureSNMPRequest struct {
	Vendor        string `json:"vendor"`
	SNMPEnabled   bool   `json:"snmp_enabled"`
	SNMPVersion   string `json:"snmp_version"`
	SNMPTarget    string `json:"snmp_target"`
	SNMPCommunity string `json:"snmp_community"`
	SNMPV3User    string `json:"snmp_v3_user"`
	SNMPAuthProto string `json:"snmp_auth_protocol"`
	SNMPAuthPass  string `json:"snmp_auth_password"`
	SNMPPrivProto string `json:"snmp_priv_protocol"`
	SNMPPrivPass  string `json:"snmp_priv_password"`
}

// listMeta segue o formato definido em api.md para paginacao.
type listMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// towersListResponse envolve a lista de torres no envelope {data, meta}
// esperado pelo antosc-front e documentado em api.md.
type towersListResponse struct {
	Data []domain.Tower `json:"data"`
	Meta listMeta       `json:"meta"`
}

func (h *TowerHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := interfaces.TowerFilter{
		Status:     r.URL.Query().Get("status"),
		OperatorID: r.URL.Query().Get("operator_id"),
		RegionID:   r.URL.Query().Get("region_id"),
		Limit:      parseIntDefault(r.URL.Query().Get("limit"), 500),
		Offset:     parseIntDefault(r.URL.Query().Get("offset"), 0),
	}

	towers, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		apierror.Internal(w)
		return
	}

	// Garante que "data" e sempre um array JSON, nunca null, mesmo sem resultados.
	if towers == nil {
		towers = []domain.Tower{}
	}

	resp := towersListResponse{
		Data: towers,
		Meta: listMeta{
			Limit:  filter.Limit,
			Offset: filter.Offset,
			Total:  total,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	// Mantido por compatibilidade com clientes que ainda leem o header,
	// mas a fonte de verdade passa a ser meta.total no corpo.
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *TowerHandler) getByID(w http.ResponseWriter, r *http.Request) {
	towerID := strings.TrimSpace(r.PathValue("id"))
	tower, err := h.service.GetByID(r.Context(), towerID)
	if err != nil {
		switch {
		case errors.Is(err, interfaces.ErrTowerNotFound), err.Error() == "tower not found":
			apierror.Write(w, http.StatusNotFound, "resource_not_found", "tower not found")
		default:
			apierror.BadRequest(w, err.Error())
		}
		return
	}

	// Disponibilidade é calculada em tempo real a partir de downtime
	// real (eventos type=failure), não um valor gravado estaticamente
	// na tabela towers — ver AvailabilityService.
	if h.availability != nil {
		if avail30, err := h.availability.Calculate(r.Context(), towerID, 30); err == nil {
			tower.Availability30d = avail30
		}
		if avail7, err := h.availability.Calculate(r.Context(), towerID, 7); err == nil {
			tower.Availability7d = &avail7
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tower)
}

func (h *TowerHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createTowerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}

	t := domain.Tower{
		ID:              strings.TrimSpace(req.ID),
		Name:            strings.TrimSpace(req.Name),
		Status:          domain.TowerStatus(strings.TrimSpace(req.Status)),
		OperatorID:      strings.TrimSpace(req.OperatorID),
		RegionID:        strings.TrimSpace(req.RegionID),
		Vendor:          strings.ToLower(strings.TrimSpace(req.Vendor)),
		SNMPEnabled:     req.SNMPEnabled,
		SNMPVersion:     strings.ToLower(strings.TrimSpace(req.SNMPVersion)),
		SNMPTarget:      strings.TrimSpace(req.SNMPTarget),
		SNMPCommunity:   strings.TrimSpace(req.SNMPCommunity),
		SNMPV3User:      strings.TrimSpace(req.SNMPV3User),
		SNMPAuthProto:   strings.ToLower(strings.TrimSpace(req.SNMPAuthProto)),
		SNMPAuthPass:    strings.TrimSpace(req.SNMPAuthPass),
		SNMPPrivProto:   strings.ToLower(strings.TrimSpace(req.SNMPPrivProto)),
		SNMPPrivPass:    strings.TrimSpace(req.SNMPPrivPass),
		Availability30d: req.Availability30d,
		CreatedAt:       time.Now().UTC(),
	}

	if err := h.service.Save(r.Context(), &t); err != nil {
		apierror.BadRequest(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(t)
}

func (h *TowerHandler) configureSNMP(w http.ResponseWriter, r *http.Request) {
	towerID := strings.TrimSpace(r.PathValue("id"))
	if towerID == "" {
		apierror.BadRequest(w, "tower id is required in path")
		return
	}

	var req configureSNMPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}

	updated, err := h.service.ConfigureSNMP(
		r.Context(),
		middleware.UserIDFromContext(r.Context()),
		towerID,
		req.Vendor,
		req.SNMPVersion,
		req.SNMPTarget,
		req.SNMPCommunity,
		req.SNMPV3User,
		req.SNMPAuthProto,
		req.SNMPAuthPass,
		req.SNMPPrivProto,
		req.SNMPPrivPass,
		req.SNMPEnabled,
	)
	if err != nil {
		apierror.BadRequest(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(updated)
}

func parseIntDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
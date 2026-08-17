// Package handlers contém implementations de handlers HTTP para varios endpoints.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"towercore/internal/core/domain"
	"towercore/internal/core/services"
)

// BackhaulInterfaceHandler lida com requisições HTTP relacionadas a métricas de interfaces de backhaul.
type BackhaulInterfaceHandler struct {
	service *services.BackhaulInterfaceService
}

// NewBackhaulInterfaceHandler cria um novo handler para métricas de backhaul.
func NewBackhaulInterfaceHandler(service *services.BackhaulInterfaceService) *BackhaulInterfaceHandler {
	return &BackhaulInterfaceHandler{service: service}
}

// ServeHTTP implementa a interface http.Handler.
func (h *BackhaulInterfaceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.collectInterface(w, r)
	case http.MethodGet:
		h.getInterfaces(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// collectInterface lida com POST /api/backhaul para receber medições de interfaces de backhaul.
// Espera um JSON object ou array de objetos BackhaulInterface no corpo da requisição.
func (h *BackhaulInterfaceHandler) collectInterface(w http.ResponseWriter, r *http.Request) {
	// Tentar decodificar como array primeiro (mais comum para batch)
	var ifaces []*domain.BackhaulInterface
	if err := json.NewDecoder(r.Body).Decode(&ifaces); err == nil {
		if len(ifaces) > 0 {
			if err := h.service.CollectMultipleInterfaces(r.Context(), ifaces); err != nil {
				http.Error(w, "failed to collect backhaul interfaces: "+err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "ok",
				"count":   strconv.Itoa(len(ifaces)),
				"message": fmt.Sprintf("collected %d backhaul interface measurements", len(ifaces)),
			})
			return
		}
	}

	// Se não for array ou estiver vazio, tentar como objeto único
	if err := json.NewDecoder(r.Body).Decode(&iface); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	iface := &domain.BackhaulInterface{}
	if err := h.service.CollectInterface(r.Context(), iface); err != nil {
		http.Error(w, "failed to collect backhaul interface: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "collected backhaul interface measurement",
	})
}

// getInterfaces lida com GET /api/backhaul para consultar medições de interfaces de backhaul.
// Suporta query parameters:
//   - tower_id: UUID da torre (obrigatório para a maioria das consultas)
//   - interface_name: nome específico da interface (opcional)
//   - limit: número máximo de resultados (padrão: 100)
//   - offset: offset para paginação (padrão: 0)
//   - order_by: campo para ordenação (padrão: measured_at desc)
//   - from: timestamp inicial (ISO 8601)
//   - to: timestamp final (ISO 8601)
//   - oper_status: filtrar por status operacional (up/down/etc)
//   - high_util: apenas interfaces com utilização > 80% (opcional)
//   - down: apenas interfaces não-up (opcional)
func (h *BackhaulInterfaceHandler) getInterfaces(w http.ResponseWriter, r *http.Request) {
	// Parse da torre_id (opcional - se não fornecido, retorna todas as torres)
	towerIDStr := r.URL.Query().Get("tower_id")
	var towerID *uuid.UUID
	if towerIDStr != "" {
		id, err := uuid.Parse(towerIDStr)
		if err != nil {
			http.Error(w, "invalid tower_id format", http.StatusBadRequest)
			return
		}
		towerID = &id
	}

	// Parse do nome da interface (opcional)
	interfaceNameStr := r.URL.Query().Get("interface_name")
	var interfaceName *string
	if interfaceNameStr != "" {
		interfaceName = &interfaceNameStr
	}

	// Parse de limite e offset
	limit := 100 // padrão
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l >= 0 {
			limit = l
		}
	}

	offset := 0 // padrão
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse de range de datas (opcional)
	var measuredAfter, measuredBefore time.Time
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			measuredAfter = t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			measuredBefore = t
		}
	}

	// Parse de filtros especiais
	highUtil := r.URL.Query().Get("high_util") == "true"
	onlyDown := r.URL.Query().Get("down") == "true"

	// Parse de order by (opcional)
	orderBy := []string{"measured_at DESC"}
	if orderByStr := r.URL.Query().Get("order_by"); orderByStr != "" {
		orderBy = []string{orderByStr}
	}

	// Construir o filtro
	filter := &domain.BackhaulInterfaceFilter{
		TowerID:     towerID,
		InterfaceName: interfaceName,
		Limit:         limit,
		Offset:        offset,
		OrderBy:       orderBy,
	}

	// Aplicar filtros de timestamp
	if !measuredAfter.IsZero() {
		filter.MeasuredAtAfter = &measuredAfter
	}
	if !measuredBefore.IsZero() {
		filter.MeasuredAtBefore = &measuredBefore
	}

	// Buscar as interfaces
	ifaces, total, err := h.service.GetInterfaceHistory(r.Context(), *towerID, *interfaceName, limit, offset, measuredAfter, measuredBefore)
	if err != nil {
		http.Error(w, "failed to retrieve backhaul interfaces: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Aplicar filtros pós-busca (mais eficiente fazer no SQL, mas vamos fazer aqui por simplicidade)
	if highUtil {
		var filtered []*domain.BackhaulInterface
		for _, iface := range ifaces {
			if iface.UtilizationPct != nil && *iface.UtilizationPct > 80.0 {
				filtered = append(filtered, iface)
			}
		}
		ifaces = filtered
	}

	if onlyDown {
		var filtered []*domain.BackhaulInterface
		for _, iface := range ifaces {
			if iface.OperStatus != nil && *iface.OperStatus != "up" {
				filtered = append(filtered, iface)
			}
		}
		ifaces = filtered
	}

	// Retornar resposta com metadados de paginação
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"count":   len(ifaces),
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"data":    ifaces,
	})
}

// GetTowerBackhaulStatus lida com GET /api/backhaul/tower/{tower_id}/status para obter
// o status consolidado de backhaul de uma torre (útil para dashboards).
func (h *BackhaulInterfaceHandler) GetTowerBackhaulStatus(w http.ResponseWriter, r *http.Request) {
	// Extrair o tower_id do path
	// Esperamos um path como: /api/backhaul/tower/{tower_id}/status
	path := r.URL.Path
	// Remover prefixo conhecido
	const prefix = "/api/backhaul/tower/"
	if !hasPrefix(path, prefix) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	towerIDStr := path[len(prefix):]
	// Remover sufixo conhecido
	const suffix = "/status"
	if !hasSuffix(towerIDStr, suffix) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	towerIDStr = towerIDStr[:len(towerIDStr)-len(suffix)]

	// Parse do tower_id
	towerID, err := uuid.Parse(towerIDStr)
	if err != nil {
		http.Error(w, "invalid tower_id in path", http.StatusBadRequest)
		return
	}

	// Buscar o status
	ifaces, err := h.service.GetTowerBackhaulStatus(r.Context(), towerID)
	if err != nil {
		http.Error(w, "failed to retrieve tower backhaul status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"count":   len(ifaces),
		"data":    ifaces,
	})
}

// Funções auxiliares para prefix/suffix
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
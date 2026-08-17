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

// SiteEnvironmentHandler lida com requisições HTTP relacionadas a medições de ambiente do site.
type SiteEnvironmentHandler struct {
	service *services.SiteEnvironmentService
}

// NewSiteEnvironmentHandler cria um novo handler para medições de ambiente.
func NewSiteEnvironmentHandler(service *services.SiteEnvironmentService) *SiteEnvironmentHandler {
	return &SiteEnvironmentHandler{service: service}
}

// ServeHTTP implementa a interface http.Handler.
func (h *SiteEnvironmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.collectEnvironment(w, r)
	case http.MethodGet:
		h.getEnvironments(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// collectEnvironment lida com POST /api/site-environment para receber medições de ambiente.
// Espera um JSON object ou array de objetos SiteEnvironment no corpo da requisição.
func (h *SiteEnvironmentHandler) collectEnvironment(w http.ResponseWriter, r *http.Request) {
	// Tentar decodificar como array primeiro (mais comum para batch)
	var envs []*domain.SiteEnvironment
	if err := json.NewDecoder(r.Body).Decode(&envs); err == nil {
		if len(envs) > 0 {
			if err := h.service.CollectMultipleEnvironment(r.Context(), envs); err != nil {
				http.Error(w, "failed to collect site environment: "+err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "ok",
				"count":   strconv.Itoa(len(envs)),
				"message": fmt.Sprintf("collected %d site environment measurements", len(envs)),
			})
			return
		}
	}

	// Se não for array ou estiver vazio, tentar como objeto único
	var env *domain.SiteEnvironment
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.CollectEnvironment(r.Context(), env); err != nil {
		http.Error(w, "failed to collect site environment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "collected site environment measurement",
	})
}

// getEnvironments lida com GET /api/site-environment para consultar medições de ambiente.
// Suporta query parameters:
//   - site_id: UUID do site (obrigatório para a maioria das consultas)
//   - limit: número máximo de resultados (padrão: 100)
//   - offset: offset para paginação (padrão: 0)
//   - order_by: campo para ordenação (padrão: measured_at desc)
//   - from: timestamp inicial (ISO 8601)
//   - to: timestamp final (ISO 8601)
//   - high_temp: apenas registros com temperatura > 35°C (opcional)
//   - high_humidity: apenas registros com umidade > 80% (opcional)
//   - door_open: apenas registros com porta aberta (opcional)
//   - mains_down: apenas registros com energia da rede ausente (opcional)
//   - ups_on_battery: apenas registros com UPS em bateria (opcional)
//   - smoke_detected: apenas registros com fumaça detectada (opcional)
func (h *SiteEnvironmentHandler) getEnvironments(w http.ResponseWriter, r *http.Request) {
	// Parse do site_id (obrigatório para a maioria das consultas)
	siteIDStr := r.URL.Query().Get("site_id")
	if siteIDStr == "" {
		http.Error(w, "site_id is required", http.StatusBadRequest)
		return
	}
	siteID, err := uuid.Parse(siteIDStr)
	if err != nil {
		http.Error(w, "invalid site_id format", http.StatusBadRequest)
		return
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
	highTemp := r.URL.Query().Get("high_temp") == "true"
	highHumidity := r.URL.Query().Get("high_humidity") == "true"
	doorOpen := r.URL.Query().Get("door_open") == "true"
	mainsDown := r.URL.Query().Get("mains_down") == "true"
	upsOnBattery := r.URL.Query().Get("ups_on_battery") == "true"
	smokeDetected := r.URL.Query().Get("smoke_detected") == "true"

	// Parse de order by (opcional)
	orderBy := []string{"measured_at DESC"}
	if orderByStr := r.URL.Query().Get("order_by"); orderByStr != "" {
		orderBy = []string{orderByStr}
	}

	// Construir o filtro
	filter := &domain.SiteEnvironmentFilter{
		SiteID:         siteID,
		Limit:          limit,
		Offset:         offset,
		OrderBy:        orderBy,
		MeasuredAtAfter: &measuredAfter,
		MeasuredAtBefore: &measuredBefore,
	}

	// Aplicar filtros especiais (como booleans)
	if highTemp {
		var tempThreshold float64 = 35.0
		filter.InternalTempCMax = &tempThreshold // na verdade queremos > 35, então vamos filtrar depois ou mudar a lógica
		// Melhor: vamos buscar tudo e filtrar no código por simplicidade (ou mudar o filtro para Min se fazia sentido)
		// Vamos ajustar: queremos TEMPERATURA ALTA, então InternalTempCMin = 35
		tempThreshold = 35.0
		filter.InternalTempCMin = &tempThreshold
	}
	if highHumidity {
		var humThreshold float64 = 80.0
		filter.HumidityPctMin = &humThreshold
	}
	if doorOpen {
		var doorOpenBool bool = true
		filter.DoorOpen = &doorOpenBool
	}
	if mainsDown {
		var mainsPowerOk bool = false
		filter.MainsPowerOk = &mainsPowerOk
	}
	if upsOnBattery {
		var upsOnBatteryBool bool = true
		filter.UpsOnBattery = &upsOnBatteryBool
	}
	if smokeDetected {
		var smokeDetectedBool bool = true
		filter.SmokeDetected = &smokeDetectedBool
	}

	// Buscar as medições
	envs, total, err := h.service.GetEnvironmentHistory(r.Context(), siteID, limit, offset, measuredAfter, measuredBefore)
	if err != nil {
		http.Error(w, "failed to retrieve site environment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Aplicar filtros pós-busca (mais simples para booleanos e ranges que não mapeiam diretamente)
	if highTemp {
		var filtered []*domain.SiteEnvironment
		for _, env := range envs {
			if env.InternalTempC != nil && *env.InternalTempC > 35.0 {
				filtered = append(filtered, env)
			}
		}
		envs = filtered
	}
	if highHumidity {
		var filtered []*domain.SiteEnvironment
		for _, env := range envs {
			if env.HumidityPct != nil && *env.HumidityPct > 80.0 {
				filtered = append(filtered, env)
			}
		}
		envs = filtered
	}
	if doorOpen {
		var filtered []*domain.SiteEnvironment
		for _, env := range envs {
			if env.DoorOpen != nil && *env.DoorOpen == true {
				filtered = append(filtered, env)
			}
		}
		envs = filtered
	}
	if mainsDown {
		var filtered []*domain.SiteEnvironment
		for _, env := range envs {
			if env.MainsPowerOk != nil && *env.MainsPowerOk == false {
				filtered = append(filtered, env)
			}
		}
		envs = filtered
	}
	if upsOnBattery {
		var filtered []*domain.SiteEnvironment
		for _, env := range envs {
			if env.UpsOnBattery != nil && *env.UpsOnBattery == true {
				filtered = append(filtered, env)
			}
		}
		envs = filtered
	}
	if smokeDetected {
		var filtered []*domain.SiteEnvironment
		for _, env := range envs {
			if env.SmokeDetected != nil && *env.SmokeDetected == true {
				filtered = append(filtered, env)
			}
		}
		envs = filtered
	}

	// Retornar resposta com metadados de paginação
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"count":   len(envs),
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"data":    envs,
	})
}

// GetSiteEnvironmentStatus lida com GET /api/site-environment/site/{site_id}/status para obter
// o status consolidado de ambiente de um site (útil para dashboards e alertas).
func (h *SiteEnvironmentHandler) GetSiteEnvironmentStatus(w http.ResponseWriter, r *http.Request) {
	// Extrair o site_id do path
	// Esperamos um path como: /api/site-environment/site/{site_id}/status
	path := r.URL.Path
	// Remover prefixo conhecido
	const prefix = "/api/site-environment/site/"
	if !hasPrefix(path, prefix) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	siteIDStr := path[len(prefix):]
	// Remover sufixo conhecido
	const suffix = "/status"
	if !hasSuffix(siteIDStr, suffix) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	siteIDStr = siteIDStr[:len(siteIDStr)-len(suffix)]

	// Parse do site_id
	siteID, err := uuid.Parse(siteIDStr)
	if err != nil {
		http.Error(w, "invalid site_id in path", http.StatusBadRequest)
		return
	}

	// Buscar o status mais recente
	env, err := h.service.GetLatestEnvironment(r.Context(), siteID)
	if err != nil {
		http.Error(w, "failed to retrieve site environment status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Retornar resposta
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"data":    env,
	})
}

// Funções auxiliares para prefix/suffix
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
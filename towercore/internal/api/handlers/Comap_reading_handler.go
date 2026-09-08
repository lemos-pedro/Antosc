package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"towercore/internal/core/interfaces"
)

// ComapReadingHandler expõe a última leitura de telemetria ComAp por torre.
// Nenhum campo é fabricado: se um valor não foi lido/confirmado, chega ao
// frontend como null (não zero), para renderizar "—" corretamente.
type ComapReadingHandler struct {
	repo interfaces.ComapReadingRepository
}

func NewComapReadingHandler(repo interfaces.ComapReadingRepository) *ComapReadingHandler {
	return &ComapReadingHandler{repo: repo}
}

// GetByTowerID trata GET /towers/{tower_id}/energy/generator
func (h *ComapReadingHandler) GetByTowerID(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")
	if towerID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "tower_id is required")
		return
	}

	reading, err := h.repo.GetByTowerID(r.Context(), towerID)
	if err != nil {
		if errors.Is(err, interfaces.ErrComapReadingNotFound) {
			writeError(w, http.StatusNotFound, "RESOURCE_NOT_FOUND", "no comap reading for this tower")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch comap reading")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(reading)
}

// writeError segue o modelo de erro padrão já documentado em api.md.
// ASSUNÇÃO: já existe um helper equivalente no pacote handlers — se
// existir, remover esta função e usar o existente para não duplicar.
// ServeHTTP permite usar ComapReadingHandler como http.Handler.
func (h *ComapReadingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetByTowerID(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

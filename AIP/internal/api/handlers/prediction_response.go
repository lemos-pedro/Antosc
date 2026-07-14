package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/antosc/aip/internal/api/dto"
	"github.com/antosc/aip/internal/prediction"
	"github.com/antosc/aip/internal/repository/postgres"
)

type PredictionHandler struct {
	predictions postgres.PredictionRepository
	service     prediction.Service
	log         *slog.Logger
}

func NewPredictionHandler(
	predictions postgres.PredictionRepository,
	service prediction.Service,
	log *slog.Logger,
) *PredictionHandler {
	return &PredictionHandler{
		predictions: predictions,
		service:     service,
		log:         log,
	}
}

// GetLatest -> GET /api/v1/predictions/{tower_id}
func (h *PredictionHandler) GetLatest(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")
	if towerID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "tower_id em falta")
		return
	}

	p, err := h.predictions.LatestByTower(r.Context(), towerID)
	if err != nil {
		h.log.Error("falha ao ler previsão", "tower_id", towerID, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "falha ao consultar previsão")
		return
	}

	if p == nil {
		writeError(w, http.StatusNotFound, "RESOURCE_NOT_FOUND", "sem previsão disponível para esta torre")
		return
	}

	resp := dto.PredictionResponse{
		TowerID:          p.TowerID,
		Model:            p.Model,
		Score:            p.Score,
		Status:           p.Status,
		Explanation:      p.Explanation,
		PredictionWindow: p.PredictionWindow,
		CreatedAt:        p.CreatedAt.UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type triggerRequest struct {
	Vendor     string `json:"vendor"`
	Model      string `json:"model"`
	WindowDays int    `json:"prediction_window_days"`
}

// Trigger -> POST /api/v1/predictions/{tower_id}
func (h *PredictionHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")
	if towerID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "tower_id em falta")
		return
	}

	var body triggerRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "payload inválido")
		return
	}
	if body.Model == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "model é obrigatório")
		return
	}
	if body.Vendor == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "vendor é obrigatório")
		return
	}
	if body.WindowDays <= 0 {
		body.WindowDays = 7
	}

	p, err := h.service.PredictAndStore(r.Context(), towerID, body.Vendor, body.Model, body.WindowDays)
	if err != nil {
		if errors.Is(err, prediction.ErrInsufficientHistory) {
			writeError(w, http.StatusConflict, "CONFLICT", err.Error())
			return
		}
		h.log.Error("falha ao gerar previsão", "tower_id", towerID, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "falha ao gerar previsão")
		return
	}

	resp := dto.PredictionResponse{
		TowerID:          p.TowerID,
		Model:            p.Model,
		Score:            p.Score,
		Status:           p.Status,
		Explanation:      p.Explanation,
		PredictionWindow: p.PredictionWindow,
		CreatedAt:        p.CreatedAt.UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(dto.ErrorResponse{
		Error: dto.ErrorBody{Code: code, Message: message},
	})
}

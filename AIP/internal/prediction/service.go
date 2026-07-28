package prediction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/antosc/aip/internal/repository/postgres"
)

// PredictRequest é o payload enviado ao serviço Python de inferência.
// O Python (python/inference) deve expor POST /predict com este contrato.
type PredictRequest struct {
    TowerID              string         `json:"tower_id"`
    Vendor               string         `json:"vendor"`
    Model                string         `json:"model"`
    PredictionWindowDays int            `json:"prediction_window_days"`
    Series               []FeaturePoint `json:"series"`
}

type FeaturePoint struct {
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// PredictResponse é o que esperamos de volta do serviço Python.
type PredictResponse struct {
	Score       float64 `json:"score"`
	Status      string  `json:"status"`
	Explanation string  `json:"explanation"`
}

// ErrInsufficientHistory sinaliza que não há dados suficientes para prever —
// regra de negócio explícita: nunca prever com histórico insuficiente.
var ErrInsufficientHistory = fmt.Errorf("histórico insuficiente para gerar previsão")

type Service interface {
	// PredictAndStore pede uma previsão ao serviço Python e persiste o
	// resultado em ai_predictions. Devolve ErrInsufficientHistory se o
	// repositório de features não tiver pontos suficientes para a torre.
	PredictAndStore(ctx context.Context, towerID, model string, windowDays int) (*postgres.Prediction, error)
}

type service struct {
	inferenceURL string
	client       *http.Client
	features     postgres.FeatureRepository
	predictions  postgres.PredictionRepository
	towers       postgres.TowerRepository

	// minHistoryPoints é o número mínimo de features históricas exigido
	// antes de pedirmos uma previsão — evita prever com base em ruído.
	minHistoryPoints int
}

func NewService(
	inferenceURL string,
	features postgres.FeatureRepository,
	predictions postgres.PredictionRepository,
	towers postgres.TowerRepository,
) Service {
	return &service{
		inferenceURL:     inferenceURL,
		client:           &http.Client{Timeout: 30 * time.Second},
		features:         features,
		predictions:      predictions,
		towers:           towers,
		minHistoryPoints: 30,
	}
}

// ErrUnknownVendor sinaliza que não sabemos o fabricante desta torre --
// sem isso o Python não sabe que adaptador usar para normalizar as métricas.
var ErrUnknownVendor = fmt.Errorf("vendor desconhecido para esta torre -- aguarda o próximo ciclo de ingestão ou confirma o tower_id")

func (s *service) PredictAndStore(ctx context.Context, towerID, model string, windowDays int) (*postgres.Prediction, error) {
	vendor, err := s.towers.GetVendor(ctx, towerID)
	if err != nil {
		return nil, err
	}
	if vendor == "" {
		return nil, ErrUnknownVendor
	}

	history, err := s.features.ListByTower(ctx, towerID, 500)
	if err != nil {
		return nil, err
	}

	if len(history) < s.minHistoryPoints {
		return nil, ErrInsufficientHistory
	}

	series := make([]FeaturePoint, 0, len(history))
	for _, f := range history {
		series = append(series, FeaturePoint{
			Name:      f.Name,
			Value:     f.Value,
			Timestamp: f.CreatedAt,
		})
	}

	reqBody, err := json.Marshal(PredictRequest{
		TowerID:              towerID,
		Vendor:                vendor,
		Model:                 model,
		PredictionWindowDays:  windowDays,
		Series:                series,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.inferenceURL+"/predict", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("serviço de inferência indisponível: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("serviço de inferência devolveu status %d", resp.StatusCode)
	}

	var out PredictResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	prediction := postgres.Prediction{
		TowerID:          towerID,
		Model:            model,
		Score:            out.Score,
		Status:           out.Status,
		Explanation:      out.Explanation,
		PredictionWindow: windowDays,
		CreatedAt:        time.Now(),
	}

	if err := s.predictions.Save(ctx, prediction); err != nil {
		return nil, err
	}

	return &prediction, nil
}

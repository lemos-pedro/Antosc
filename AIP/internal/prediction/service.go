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
	//
	// vendor identifica o fabricante do equipamento da torre (eltek,
	// enetek, huawei) e é obrigatório: o serviço de inferência usa-o para
	// escolher o adaptador de normalização correto (vendors/registry.py).
	// O AIP ainda não tem uma fonte própria de "vendor por torre" (nem o
	// towercore expõe isso hoje em TowerDTO), por isso é o chamador que
	// tem de o indicar explicitamente -- nunca assumimos um vendor por
	// omissão.
	PredictAndStore(ctx context.Context, towerID, vendor, model string, windowDays int) (*postgres.Prediction, error)
}

type service struct {
	inferenceURL string
	client       *http.Client
	features     postgres.FeatureRepository
	predictions  postgres.PredictionRepository

	// minHistoryPoints é o número mínimo de features históricas exigido
	// antes de pedirmos uma previsão — evita prever com base em ruído.
	minHistoryPoints int
}

func NewService(
	inferenceURL string,
	features postgres.FeatureRepository,
	predictions postgres.PredictionRepository,
) Service {
	return &service{
		inferenceURL:     inferenceURL,
		client:           &http.Client{Timeout: 30 * time.Second},
		features:         features,
		predictions:      predictions,
		minHistoryPoints: 30,
	}
}

func (s *service) PredictAndStore(ctx context.Context, towerID, vendor, model string, windowDays int) (*postgres.Prediction, error) {
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
		Vendor:               vendor,
		Model:                model,
		PredictionWindowDays: windowDays,
		Series:               series,
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

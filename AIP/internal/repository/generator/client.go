// Package generator fala com o serviço ComAp de energia do gerador —
// serviço separado do towercore (porta própria), por isso tem o seu
// próprio cliente em vez de ser mais um método em towercore.Client.
package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// EnergyDTO espelha o payload de GET /api/v1/towers/{tower_id}/energy/generator.
type EnergyDTO struct {
	TowerID         string    `json:"TowerID"`
	FuelLiters      float64   `json:"FuelLiters"`
	FuelPercent     float64   `json:"FuelPercent"`
	BatteryVoltageV float64   `json:"BatteryVoltageV"`
	RunHoursTotal   float64   `json:"RunHoursTotal"`
	CollectedAt     time.Time `json:"CollectedAt"`
}

// Client é a porta de saída do AIP para o serviço de energia do gerador.
// Só leitura, tal como towercore.Client.
type Client interface {
	GetEnergy(ctx context.Context, towerID string) (EnergyDTO, error)
}

type httpClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string) Client {
	return &httpClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *httpClient) GetEnergy(ctx context.Context, towerID string) (EnergyDTO, error) {
	url := fmt.Sprintf("%s/api/v1/towers/%s/energy/generator", c.baseURL, towerID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return EnergyDTO{}, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return EnergyDTO{}, fmt.Errorf("generator GetEnergy: %w", err)
	}
	defer resp.Body.Close()

	// 404 é esperado para sites sem gerador reportado (ex: só retificador) --
	// não é um erro do sistema, só significa "sem dados de gerador aqui".
	if resp.StatusCode == http.StatusNotFound {
		return EnergyDTO{}, ErrNoGenerator
	}
	if resp.StatusCode != http.StatusOK {
		return EnergyDTO{}, fmt.Errorf("status inesperado %d em %s", resp.StatusCode, url)
	}

	var out EnergyDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return EnergyDTO{}, err
	}
	return out, nil
}

// ErrNoGenerator sinaliza que o site não tem gerador reportado -- distinto
// de um erro de rede/serviço, para a ingestão poder ignorar sem barulho.
var ErrNoGenerator = fmt.Errorf("site sem gerador reportado")

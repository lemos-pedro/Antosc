package towercore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (c *HTTPClient) GetTowers(ctx context.Context) ([]TowerDTO, error) {
	var envelope struct {
		Data []TowerDTO `json:"data"`
	}

	if err := c.getJSON(ctx, "/api/v1/towers?limit=200", &envelope); err != nil {
		return nil, fmt.Errorf("towercore GetTowers: %w", err)
	}

	return envelope.Data, nil
}

// GetEvents agrega eventos de todas as torres (api.md só expõe eventos por
// torre: GET /towers/{tower_id}/events). Para fleets grandes isto é N+1 —
// aceitável para já dado o volume (~212 sites), mas é candidato a um
// endpoint agregado (GET /events) se o volume crescer.
func (c *HTTPClient) GetEvents(ctx context.Context) ([]EventDTO, error) {
	// NOTA: ao contrário de GetTowers, este endpoint do towercore devolve
	// o array diretamente (sem envelope {"data":...}) -- confirmado em
	// produção (erro de unmarshal ao tentar decodificar como envelope).
	// Mesmo padrão inconsistente já documentado em GetMetrics.
	var events []EventDTO

	if err := c.getJSON(
		ctx,
		"/api/v1/events",
		&events,
	); err != nil {
		return nil, fmt.Errorf("towercore GetEvents: %w", err)
	}

	return events, nil
}

// GetMetrics lê GET /api/v1/metrics. Este endpoint devolve o array
// diretamente (sem envelope {"data":...}) e o total num header
// X-Total-Count, ao contrário do resto da API — ver MetricDTO.
//
// since filtra por "from" para o scheduler só puxar o que mudou desde o
// último ciclo, em vez do histórico completo a cada 5 minutos.
// Pagina automaticamente enquanto o towercore devolver o número máximo
// de linhas pedido (assume-se que há mais páginas).
func (c *HTTPClient) GetMetrics(ctx context.Context, since time.Time) ([]MetricDTO, error) {
	const pageSize = 200

	var all []MetricDTO
	offset := 0

	for {
		path := fmt.Sprintf(
			"/api/v1/metrics?from=%s&limit=%d&offset=%d",
			since.UTC().Format(time.RFC3339),
			pageSize,
			offset,
		)

		var page []MetricDTO
		if err := c.getJSON(ctx, path, &page); err != nil {
			return nil, fmt.Errorf("towercore GetMetrics: %w", err)
		}

		all = append(all, page...)

		if len(page) < pageSize {
			break
		}
		offset += pageSize
	}

	return all, nil
}

func (c *HTTPClient) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status inesperado %d em %s", resp.StatusCode, path)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

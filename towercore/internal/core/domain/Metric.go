package domain

import "time"

// Metric representa uma amostra de dados recolhida de um equipamento numa torre.
type Metric struct {
	ID          string             `json:"metric_id"`
	TowerID     string             `json:"tower_id"`
	CollectedAt time.Time          `json:"collected_at"`
	Values      map[string]float64 `json:"metrics"`
	CreatedAt   time.Time          `json:"created_at"`
}

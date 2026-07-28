package dto

import "time"

// ConsumptionNormRequest é o payload que o Controller envia para criar/atualizar uma norma.
type ConsumptionNormRequest struct {
	TowerID          string  `json:"tower_id"`
	EquipmentType    string  `json:"equipment_type"`
	ExpectedValue    float64 `json:"expected_value"`
	Unit             string  `json:"unit"`
	TolerancePercent float64 `json:"tolerance_percent"`
	DefinedBy        string  `json:"defined_by"`
}

type ConsumptionNormResponse struct {
	ID               string    `json:"id"`
	TowerID          string    `json:"tower_id"`
	EquipmentType    string    `json:"equipment_type"`
	ExpectedValue    float64   `json:"expected_value"`
	Unit             string    `json:"unit"`
	TolerancePercent float64   `json:"tolerance_percent"`
	DefinedBy        string    `json:"defined_by"`
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

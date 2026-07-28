package dto

type ConsumptionDeviationResponse struct {
	TowerID          string  `json:"tower_id"`
	EquipmentType    string  `json:"equipment_type"`
	ExpectedValue    float64 `json:"expected_value"`
	AverageActual    float64 `json:"average_actual"`
	Unit             string  `json:"unit"`
	TolerancePercent float64 `json:"tolerance_percent"`
	DeviationPercent float64 `json:"deviation_percent"`
	WithinNorm       bool    `json:"within_norm"`
	SampleCount      int     `json:"sample_count"`
}

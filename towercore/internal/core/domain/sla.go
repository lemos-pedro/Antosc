package domain

// SLAReport representa o relatório consolidado de disponibilidade.
type SLAReport struct {
	WindowDays          int     `json:"window_days"`
	AvailabilityPercent float64 `json:"availability_percent"`
	AffectedTowers      int     `json:"affected_towers"`
}

// SLAThreshold define o limiar abaixo do qual uma torre está em violação de SLA.
const SLAThreshold = 99.5

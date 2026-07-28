package dto

import "time"

// IncidentCauseConfirmRequest é o payload que o O&M envia para confirmar/corrigir uma causa.
type IncidentCauseConfirmRequest struct {
	ConfirmedCause string `json:"confirmed_cause"`
	ConfirmedBy    string `json:"confirmed_by"`
}

type IncidentCauseResponse struct {
	ID                string     `json:"id"`
	TowerID           string     `json:"tower_id"`
	EventID           *string    `json:"event_id,omitempty"`
	IncidentStartedAt time.Time  `json:"incident_started_at"`
	IncidentEndedAt   *time.Time `json:"incident_ended_at,omitempty"`
	MLPredictedCause  *string    `json:"ml_predicted_cause,omitempty"`
	MLConfidence      *float64   `json:"ml_confidence,omitempty"`
	ConfirmedCause    *string    `json:"confirmed_cause,omitempty"`
	ConfirmedBy       *string    `json:"confirmed_by,omitempty"`
	ConfirmedAt       *time.Time `json:"confirmed_at,omitempty"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
}

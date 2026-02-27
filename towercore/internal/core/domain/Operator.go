package domain

import "time"

// Operator representa uma empresa operadora hospedada numa torre.
type Operator struct {
	ID        string    `json:"operator_id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

package domain

import "time"

// Region representa uma zona geográfica que agrupa torres.
type Region struct {
	ID        string    `json:"region_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

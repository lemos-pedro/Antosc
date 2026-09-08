package domain

import "time"

type Region struct {
	RegionID  string    `json:"region_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

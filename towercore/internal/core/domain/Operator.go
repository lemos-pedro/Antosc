package domain

import "time"

type Operator struct {
	OperatorID string    `json:"operator_id"`
	Name       string    `json:"name"`
	Code       string    `json:"code"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
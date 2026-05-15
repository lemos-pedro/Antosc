package domain

import "time"

type AuditLog struct {
	ID         string    `json:"audit_id"`
	Actor      string    `json:"actor"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resource_id"`
	Details    string    `json:"details"`
	CreatedAt  time.Time `json:"created_at"`
}

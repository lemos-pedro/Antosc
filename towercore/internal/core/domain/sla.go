package domain

type SLA struct {
	Availability float64 `json:"availability"`
	TotalSites   int     `json:"total_sites"`
	OnlineSites  int     `json:"online_sites"`
	OfflineSites int     `json:"offline_sites"`
	DegradedSites int    `json:"degraded_sites"`
}
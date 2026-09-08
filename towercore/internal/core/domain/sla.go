package domain

type SLA struct {
	RegionID      string  `json:"region_id,omitempty"`
	RegionName    string  `json:"region_name,omitempty"`
	Availability  float64 `json:"availability"`
	TotalSites    int     `json:"total_sites"`
	OnlineSites   int     `json:"online_sites"`
	OfflineSites  int     `json:"offline_sites"`
	DegradedSites int     `json:"degraded_sites"`
}

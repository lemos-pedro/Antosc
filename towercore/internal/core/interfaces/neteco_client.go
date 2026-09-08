package interfaces

// BatteryStatus representa o estado da bateria de um site, obtido via NetEco.
type BatteryStatus struct {
	SiteDn        string
	SiteName      string
	ConnectStatus string
	SOC           float64 // -1 = sem dado
	SOH           float64 // -1 = sem dado
	BackupTimeH   float64
}

// SiteCounterInfo representa contadores de energia DC de um site, obtidos via NetEco.
type SiteCounterInfo struct {
	DCOutputVoltage  *float64
	DCLoadCurrent    *float64
	RectifierCurrent *float64
}

// NetEcoClient define o contrato de acesso ao NetEco (Huawei NBI/REST interno).
type NetEcoClient interface {
	Login() error
	FetchBatteryStatus(siteDn string) (*BatteryStatus, error)
	FetchSiteCounterInfo(siteDn string) (*SiteCounterInfo, error)
}

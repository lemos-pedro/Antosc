package interfaces

type BatteryStatus struct {
	SiteDn        string
	SiteName      string
	ConnectStatus string
	SOC           float64
	SOH           float64
	BackupTimeH   float64
}

type NetEcoClient interface {
	Login() error
	FetchBatteryStatus(siteDn string) (*BatteryStatus, error)
}
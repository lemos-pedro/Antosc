package hizima

// envelope é o wrapper comum de sucesso das 3 respostas.
type envelope[T any] struct {
	Code    int    `json:"code"`
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    pageDTO[T] `json:"data"`
}

// failureEnvelope é o formato de erro documentado (status como string "failed").
type failureEnvelope struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type pageDTO[T any] struct {
	Records []T `json:"records"`
	Total   int `json:"total"`
	Size    int `json:"size"`
	Current int `json:"current"`
	Pages   int `json:"pages"`
}

// lockStatusDTO espelha um record de GET /clientExchangeApi/lock/statusBySite.
type lockStatusDTO struct {
	LockID          int64   `json:"lockId"`
	RegionName1     string  `json:"regionName1"`
	RegionName2     string  `json:"regionName2"`
	RegionName3     string  `json:"regionName3"`
	StationName     string  `json:"stationName"`
	StationNo       string  `json:"stationNo"`
	CustomerSiteID  string  `json:"customerSiteId"`
	LockName        string  `json:"lockName"`
	DeviceID        string  `json:"deviceId"`
	OpenState       *int    `json:"openState"` // null possível -> Unknown
	Battery         *int    `json:"battery"`
	BatteryTime     string  `json:"batteryTime"`   // "2026-08-30 14:05:00", server local time
	LastOperTime    string  `json:"lastOperTime"`
	LastSyncTime    string  `json:"lastSyncTime"`
	FirmwareVersion string  `json:"firmwareVersion"`
	HardwareModel   string  `json:"hardwareModel"`
}

// lockEventDTO espelha um record de GET /clientExchangeApi/lockOpenLog/page.
type lockEventDTO struct {
	ID               int64  `json:"id"`
	DeviceID         string `json:"deviceId"`
	LockName         string `json:"lockName"`
	StationName      string `json:"stationName"`
	StationNo        string `json:"stationNo"`
	EventType        int    `json:"eventType"`
	OperatorRealName string `json:"operatorRealName"`
	OperatorAccount  string `json:"operatorAccount"`
	OperTime         string `json:"operTime"`    // server local time
	OperTimeIso      string `json:"operTimeIso"` // com offset
	OperTimeUtc      string `json:"operTimeUtc"` // preferir este para armazenamento interno
	TicketUID        string `json:"ticketUid"`
}

// workOrderDTO espelha um record de GET /clientExchangeApi/ticket/statusPage.
type workOrderDTO struct {
	TicketID         int64  `json:"ticketId"`
	TicketUID        string `json:"ticketUid"`
	SiteName         string `json:"siteName"`
	SiteNo           string `json:"siteNo"`
	CustomerSiteID   string `json:"customerSiteId"`
	ApplicantAccount string `json:"applicantAccount"`
	ApplicantName    string `json:"applicantName"`
	Status           int    `json:"status"`
	CreateTime       string `json:"createTime"`
	CreateTimeUtc    string `json:"createTimeUtc"`
	ApproveTime      string `json:"approveTime"`
	ApproveTimeUtc   string `json:"approveTimeUtc"`
	FinishTime       *string `json:"finishTime"`
	FinishTimeUtc    *string `json:"finishTimeUtc"`
}
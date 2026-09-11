package domain

import "time"

// LockState representa o estado físico do cadeado.
type LockState int

const (
	LockStateUnknown  LockState = iota // null no CMS — sem comunicação recente
	LockStateUnlocked                  // openState: 0
	LockStateLocked                    // openState: 1
	LockStateFault                     // openState: 2
)

func (s LockState) String() string {
	switch s {
	case LockStateUnlocked:
		return "unlocked"
	case LockStateLocked:
		return "locked"
	case LockStateFault:
		return "fault"
	default:
		return "unknown"
	}
}

// Lock representa o estado consolidado de um cadeado ZMACS associado a uma torre.
// Mapeado a partir de GET /clientExchangeApi/lock/statusBySite.
type Lock struct {
	LockID          int64
	TowerID         string // resolvido internamente via cross-reference (StationNo/CustomerSiteID -> tower_id)
	StationNo       string // sno — ID externo estável, chave de cross-reference primária
	CustomerSiteID  string // ID externo alternativo
	StationName     string
	RegionPath      [3]string // regionName1/2/3, do mais amplo ao mais específico
	LockName        string
	DeviceID        string // hardware ID, pode servir de chave alternativa
	State           LockState
	BatteryPercent  *int // ponteiro: nil quando não reportado, nunca fabricar valor
	BatteryTime     *time.Time
	LastOperTime    *time.Time
	LastSyncTime    *time.Time // último sync real do cadeado BLE via app do técnico
	FirmwareVersion string
	HardwareModel   string
}

// IsStale indica se o cadeado não sincroniza há mais que o threshold dado.
// Útil porque cadeados BLE só reportam quando o técnico sincroniza a app —
// "unknown" prolongado é operacionalmente distinto de "fault".
func (l Lock) IsStale(threshold time.Duration, now time.Time) bool {
	if l.LastSyncTime == nil {
		return true
	}
	return now.Sub(*l.LastSyncTime) > threshold
}

// LockEventType representa o tipo de operação registada no log de acessos.
type LockEventType int

const (
	LockEventUnknown LockEventType = iota
	LockEventUnlock
	LockEventLock
)

func (t LockEventType) String() string {
	switch t {
	case LockEventUnlock:
		return "unlock"
	case LockEventLock:
		return "lock"
	default:
		return "unknown"
	}
}

// LockEvent representa uma entrada no histórico de acessos.
// Mapeado a partir de GET /clientExchangeApi/lockOpenLog/page.
type LockEvent struct {
	LogID            int64
	DeviceID         string
	LockName         string
	StationName      string
	StationNo        string
	TowerID          string // resolvido internamente
	EventType        LockEventType
	OperatorRealName string
	OperatorAccount  string
	OccurredAt       time.Time // operTimeUtc, já em UTC
	WorkOrderUID     string    // ticketUid — vazio se não associado a work order
}

// WorkOrderStatus representa o estado de um pedido de acesso.
type WorkOrderStatus int

const (
	WorkOrderPending WorkOrderStatus = iota
	WorkOrderApproved
	WorkOrderCompleted
	WorkOrderExpired
	WorkOrderRejected
	WorkOrderTerminated
)

func (s WorkOrderStatus) String() string {
	switch s {
	case WorkOrderPending:
		return "pending"
	case WorkOrderApproved:
		return "approved"
	case WorkOrderCompleted:
		return "completed"
	case WorkOrderExpired:
		return "expired"
	case WorkOrderRejected:
		return "rejected"
	case WorkOrderTerminated:
		return "terminated"
	default:
		return "unknown"
	}
}

// WorkOrder representa um pedido de acesso a um site.
// Mapeado a partir de GET /clientExchangeApi/ticket/statusPage.
type WorkOrder struct {
	TicketID          int64
	TicketUID         string
	SiteName          string
	SiteNo            string
	CustomerSiteID    string
	TowerID           string // resolvido internamente
	ApplicantAccount  string
	ApplicantName     string
	Status            WorkOrderStatus
	CreatedAt         time.Time
	ApprovedAt        *time.Time
	FinishedAt        *time.Time
}
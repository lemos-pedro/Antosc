package interfaces

import (
	"context"

	"towercore/internal/core/domain"
)

// LockStatusFilter espelha os query params opcionais de statusBySite.
type LockStatusFilter struct {
	StationNo  string // sno — filtra por site externo
	StationID  *int64 // internal station ID
	LockName   string // fuzzy match
	DeviceID   string // fuzzy match
	OpenState  *domain.LockState
}

// LockEventFilter espelha os query params opcionais de lockOpenLog/page.
type LockEventFilter struct {
	TicketID     *int64
	TicketUID    string
	LockID       *int64
	LockName     string
	DeviceID     string
	StationID    *int64
	StationName  string
	StationNo    string
	OperatorID   *int64
	OperatorName string
	RealName     string
	EventType    *domain.LockEventType
	StartTime    string // formato aceite pela API — passar como veio, sem reformatar
	EndTime      string
}

// WorkOrderFilter espelha os query params opcionais de ticket/statusPage.
type WorkOrderFilter struct {
	UID          string
	StationID    *int64
	StationName  string
	AuthResult   *int
	ApplicantAcc string
	StartTime    string
	EndTime      string
}

// PageRequest controla a paginação comum aos 3 endpoints.
type PageRequest struct {
	Current int // 1-indexed, obrigatório pela API
	Size    int // obrigatório pela API
}

// Page é o envelope de paginação devolvido pelo CMS.
type Page[T any] struct {
	Records []T
	Total   int
	Size    int
	Current int
	Pages   int
}

// AccessControlPort é o contrato que o adapter Hizima implementa.
// Nota: propositadamente só de leitura — o CMS ZMACS não expõe
// trancar/destrancar remotamente; essa ação é exclusiva da app
// mobile do técnico via BLE. O TowerCore consulta e audita, não comanda.
type AccessControlPort interface {
	GetLockStatus(ctx context.Context, filter LockStatusFilter, page PageRequest) (Page[domain.Lock], error)
	GetLockEvents(ctx context.Context, filter LockEventFilter, page PageRequest) (Page[domain.LockEvent], error)
	GetWorkOrders(ctx context.Context, filter WorkOrderFilter, page PageRequest) (Page[domain.WorkOrder], error)
}
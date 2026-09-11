package hizima

import (
	"time"

	"towercore/internal/core/domain"
)

// Layout usado pelo CMS para timestamps em server-local-time.
// Ex.: "2026-08-30 14:05:00". Não confundir com os campos *Utc,
// que já vêm em RFC3339 e devem ser preferidos quando existirem.
const cmsLocalLayout = "2006-01-02 15:04:05"

func parseCMSLocalTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(cmsLocalLayout, s)
	if err != nil {
		return nil // nunca fabricar valor — antes nil que um timestamp errado
	}
	return &t
}

func parseUTC(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func mapOpenState(openState *int) domain.LockState {
	if openState == nil {
		return domain.LockStateUnknown
	}
	switch *openState {
	case 0:
		return domain.LockStateUnlocked
	case 1:
		return domain.LockStateLocked
	case 2:
		return domain.LockStateFault
	default:
		return domain.LockStateUnknown
	}
}

func mapEventType(eventType int) domain.LockEventType {
	switch eventType {
	case 0:
		return domain.LockEventUnlock
	case 1:
		return domain.LockEventLock
	default:
		return domain.LockEventUnknown
	}
}

func mapWorkOrderStatus(status int) domain.WorkOrderStatus {
	switch status {
	case 0:
		return domain.WorkOrderPending
	case 1:
		return domain.WorkOrderApproved
	case 2:
		return domain.WorkOrderCompleted
	case 3:
		return domain.WorkOrderExpired
	case 4:
		return domain.WorkOrderRejected
	case 5:
		return domain.WorkOrderTerminated
	default:
		return domain.WorkOrderPending
	}
}

// toDomainLock converte o DTO para o domínio. TowerID fica vazio aqui
// de propósito — a resolução StationNo/CustomerSiteID -> tower_id é
// responsabilidade da camada de serviço (core/services), não do adapter,
// para manter o adapter livre de dependências do resto do schema.
func toDomainLock(d lockStatusDTO) domain.Lock {
	return domain.Lock{
		LockID:          d.LockID,
		StationNo:       d.StationNo,
		CustomerSiteID:  d.CustomerSiteID,
		StationName:     d.StationName,
		RegionPath:      [3]string{d.RegionName1, d.RegionName2, d.RegionName3},
		LockName:        d.LockName,
		DeviceID:        d.DeviceID,
		State:           mapOpenState(d.OpenState),
		BatteryPercent:  d.Battery,
		BatteryTime:     parseCMSLocalTime(d.BatteryTime),
		LastOperTime:    parseCMSLocalTime(d.LastOperTime),
		LastSyncTime:    parseCMSLocalTime(d.LastSyncTime),
		FirmwareVersion: d.FirmwareVersion,
		HardwareModel:   d.HardwareModel,
	}
}

func toDomainLockEvent(d lockEventDTO) domain.LockEvent {
	occurredAt, ok := parseUTC(d.OperTimeUtc)
	if !ok {
		// fallback: sem UTC parseável, usar local time se existir
		if lt := parseCMSLocalTime(d.OperTime); lt != nil {
			occurredAt = *lt
		}
	}
	return domain.LockEvent{
		LogID:            d.ID,
		DeviceID:         d.DeviceID,
		LockName:         d.LockName,
		StationName:      d.StationName,
		StationNo:        d.StationNo,
		EventType:        mapEventType(d.EventType),
		OperatorRealName: d.OperatorRealName,
		OperatorAccount:  d.OperatorAccount,
		OccurredAt:       occurredAt,
		WorkOrderUID:     d.TicketUID,
	}
}

func toDomainWorkOrder(d workOrderDTO) domain.WorkOrder {
	wo := domain.WorkOrder{
		TicketID:         d.TicketID,
		TicketUID:        d.TicketUID,
		SiteName:         d.SiteName,
		SiteNo:           d.SiteNo,
		CustomerSiteID:   d.CustomerSiteID,
		ApplicantAccount: d.ApplicantAccount,
		ApplicantName:    d.ApplicantName,
		Status:           mapWorkOrderStatus(d.Status),
	}
	if t, ok := parseUTC(d.CreateTimeUtc); ok {
		wo.CreatedAt = t
	} else if lt := parseCMSLocalTime(d.CreateTime); lt != nil {
		wo.CreatedAt = *lt
	}
	if d.ApproveTimeUtc != "" {
		if t, ok := parseUTC(d.ApproveTimeUtc); ok {
			wo.ApprovedAt = &t
		}
	}
	if d.FinishTimeUtc != nil && *d.FinishTimeUtc != "" {
		if t, ok := parseUTC(*d.FinishTimeUtc); ok {
			wo.FinishedAt = &t
		}
	}
	return wo
}
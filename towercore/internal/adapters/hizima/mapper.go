package hizima

import (
	"time"

	"towercore/internal/core/domain"
)

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
// responsabilidade da camada de serviço (core/services), não do adapter.
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
		BatteryTime:     d.BatteryTime.Time(),
		LastOperTime:    d.LastOperTime.Time(),
		LastSyncTime:    d.LastSyncTime.Time(),
		FirmwareVersion: d.FirmwareVersion,
		HardwareModel:   d.HardwareModel,
	}
}

func toDomainLockEvent(d lockEventDTO) domain.LockEvent {
	// Preferir OperTimeUtc; se vier nil, tentar OperTimeIso, depois OperTime.
	var occurredAt time.Time
	if t := d.OperTimeUtc.Time(); t != nil {
		occurredAt = *t
	} else if t := d.OperTimeIso.Time(); t != nil {
		occurredAt = *t
	} else if t := d.OperTime.Time(); t != nil {
		occurredAt = *t
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
	if t := d.CreateTimeUtc.Time(); t != nil {
		wo.CreatedAt = *t
	} else if t := d.CreateTime.Time(); t != nil {
		wo.CreatedAt = *t
	}
	if t := d.ApproveTimeUtc.Time(); t != nil {
		wo.ApprovedAt = t
	} else if t := d.ApproveTime.Time(); t != nil {
		wo.ApprovedAt = t
	}
	if t := d.FinishTimeUtc.Time(); t != nil {
		wo.FinishedAt = t
	} else if t := d.FinishTime.Time(); t != nil {
		wo.FinishedAt = t
	}
	return wo
}
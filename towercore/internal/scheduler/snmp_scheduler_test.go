package scheduler

import (
	"context"
	"testing"
	"time"

	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/internal/infrastructure/logger"
)

func TestSNMPScheduler_CollectOnce(t *testing.T) {
	towerRepo := &fakeTowerRepo{
		towers: []domain.Tower{
			{
				ID:          "00000000-0000-0000-0000-000000000001",
				Status:      domain.TowerStatusOnline,
				Vendor:      "eltek",
				SNMPEnabled: true,
			},
		},
	}

	metricRepo := &fakeMetricRepo{}
	eventRepo := &fakeEventRepo{}
	metricSvc := services.NewMetricService(metricRepo)
	eventSvc := services.NewEventService(eventRepo)

	ingestSvc := services.NewSNMPIngestService(metricSvc, eventSvc, map[string]snmp.Profile{
		"eltek": {
			Vendor: "eltek",
			Metrics: []snmp.MetricDefinition{
				{OID: ".1.3.6.1.4.1.12148.10.10.5.5.0", Key: "battery_voltage_v", Scale: 0.01},
			},
			Alarms: []snmp.AlarmRule{
				{Key: "battery_voltage_v", Threshold: 20, Condition: "gt", Severity: domain.EventSeverityWarning, Message: "voltage high"},
			},
		},
	})

	s := NewSNMPScheduler(
		towerRepo,
		ingestSvc,
		snmp.NewSyntheticCollector(),
		map[string]snmp.Profile{
			"eltek": {
				Vendor: "eltek",
				Metrics: []snmp.MetricDefinition{
					{OID: ".1.3.6.1.4.1.12148.10.10.5.5.0", Key: "battery_voltage_v", Scale: 0.01},
				},
			},
		},
		logger.New("info"),
		time.Second,
		10,
	)

	s.CollectOnce(context.Background())

	if metricRepo.created == 0 {
		t.Fatalf("expected metric to be created")
	}
	if eventRepo.created == 0 {
		t.Fatalf("expected alarm event to be created")
	}
}

type fakeTowerRepo struct {
	towers []domain.Tower
}

func (r *fakeTowerRepo) List(_ context.Context, _ interfaces.TowerFilter) ([]domain.Tower, int, error) {
	return r.towers, len(r.towers), nil
}
func (r *fakeTowerRepo) GetByID(_ context.Context, _ string) (*domain.Tower, error) { return nil, nil }
func (r *fakeTowerRepo) Upsert(_ context.Context, _ *domain.Tower) error            { return nil }

type fakeMetricRepo struct{ created int }

func (r *fakeMetricRepo) Create(_ context.Context, _ *domain.Metric) error {
	r.created++
	return nil
}

func (r *fakeMetricRepo) List(_ context.Context, _ interfaces.MetricFilter) ([]domain.Metric, int, error) {
	return nil, 0, nil
}

type fakeEventRepo struct{ created int }

func (r *fakeEventRepo) Create(_ context.Context, _ *domain.Event) error {
	r.created++
	return nil
}

func (r *fakeEventRepo) List(_ context.Context, _ interfaces.EventFilter) ([]domain.Event, int, error) {
	return nil, 0, nil
}

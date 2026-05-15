package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"towercore/internal/adapters/eltek"
	"towercore/internal/adapters/enetek"
	"towercore/internal/adapters/huawei"
	"towercore/internal/adapters/mock"
	"towercore/internal/adapters/snmp"
	"towercore/internal/core/services"
)

func TestSNMPCollectHandler_ServeHTTP(t *testing.T) {
	eventSvc := services.NewEventService(mock.NewEventRepository())
	metricSvc := services.NewMetricService(mock.NewMetricRepository())
	profiles := map[string]snmp.Profile{
		"eltek":  eltek.Profile(),
		"huawei": huawei.Profile(),
		"enetek": enetek.Profile(),
	}
	handler := NewSNMPCollectHandler(services.NewSNMPIngestService(metricSvc, eventSvc, profiles))

	payload := []byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","vendor":"eltek","samples":{".1.3.6.1.4.1.12148.10.10.5.5.0":4850}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collect/snmp", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestSNMPCollectHandler_BadRequest(t *testing.T) {
	eventSvc := services.NewEventService(mock.NewEventRepository())
	metricSvc := services.NewMetricService(mock.NewMetricRepository())
	profiles := map[string]snmp.Profile{
		"eltek":  eltek.Profile(),
		"huawei": huawei.Profile(),
		"enetek": enetek.Profile(),
	}
	handler := NewSNMPCollectHandler(services.NewSNMPIngestService(metricSvc, eventSvc, profiles))

	payload := []byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","vendor":"unknown","samples":{"1.2.3":1}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collect/snmp", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

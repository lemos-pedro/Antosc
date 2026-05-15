package routes

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"towercore/internal/adapters/eltek"
	"towercore/internal/adapters/enetek"
	"towercore/internal/adapters/huawei"
	"towercore/internal/adapters/mock"
	"towercore/internal/adapters/snmp"
	"towercore/internal/api/handlers"
	"towercore/internal/core/services"
	"towercore/internal/infrastructure/logger"
)

func TestSmoke_Flow(t *testing.T) {
	cfg := testConfig()
	log := logger.New("info")

	towerRepo := mock.NewTowerRepository()
	towerHandler := handlers.NewTowerHandler(services.NewTowerService(towerRepo, mock.NewAuditRepository()))
	eventSvc := services.NewEventService(mock.NewEventRepository())
	metricSvc := services.NewMetricService(mock.NewMetricRepository())
	auditSvc := services.NewAuditService(mock.NewAuditRepository())
	eventHandler := handlers.NewEventHandler(eventSvc)
	metricHandler := handlers.NewMetricHandler(metricSvc)
	auditHandler := handlers.NewAuditHandler(auditSvc)
	snmpProfiles := map[string]snmp.Profile{
		"eltek":  eltek.Profile(),
		"huawei": huawei.Profile(),
		"enetek": enetek.Profile(),
	}
	snmpCollectHandler := handlers.NewSNMPCollectHandler(services.NewSNMPIngestService(metricSvc, eventSvc, snmpProfiles))
	authRepo := mock.NewUserRepository()
	authSvc := services.NewAuthService(authRepo, cfg.Auth.UserTokenSecret, time.Hour)
	_ = authSvc.EnsureBootstrapUser(context.Background(), "admin", "admin123", "admin")
	authHandler := handlers.NewAuthHandler(authSvc)

	router := NewRouter(cfg, log, towerHandler, eventHandler, metricHandler, snmpCollectHandler, auditHandler, authHandler)

	assertStatus(t, router, http.MethodGet, "/health", nil, http.StatusOK)
	assertStatus(t, router, http.MethodPost, "/api/v1/auth/login", []byte(`{"username":"admin","password":"admin123"}`), http.StatusOK)
	assertStatus(t, router, http.MethodPost, "/api/v1/towers", []byte(`{"name":"Tower-SMOKE-01","status":"online"}`), http.StatusCreated)
	assertStatus(t, router, http.MethodGet, "/api/v1/towers?status=online&limit=10&offset=0", nil, http.StatusOK)
	assertStatus(t, router, http.MethodGet, "/api/v1/towers/00000000-0000-0000-0000-000000000001", nil, http.StatusOK)
	assertStatus(t, router, http.MethodPost, "/api/v1/events", []byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","type":"alarm","severity":"warning","message":"smoke test"}`), http.StatusCreated)
	assertStatus(t, router, http.MethodPost, "/api/v1/metrics", []byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","metrics":{"voltage":48.4}}`), http.StatusCreated)
	assertStatus(t, router, http.MethodGet, "/api/v1/events?limit=10&offset=0", nil, http.StatusOK)
	assertStatus(t, router, http.MethodGet, "/api/v1/metrics?limit=10&offset=0", nil, http.StatusOK)
	assertStatus(t, router, http.MethodPost, "/api/v1/collect/snmp", []byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","vendor":"eltek","samples":{".1.3.6.1.4.1.12148.10.10.5.5.0":4850}}`), http.StatusAccepted)
	assertStatus(t, router, http.MethodGet, "/api/v1/audit-logs?limit=10&offset=0", nil, http.StatusOK)
	assertStatus(t, router, http.MethodGet, "/api/v1/audit-logs/export.csv", nil, http.StatusOK)
}

func assertStatus(t *testing.T, router http.Handler, method, path string, body []byte, expected int) {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet || pathHasAuthRead(path) {
		authorizeWrite(req)
	}
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != expected {
		t.Fatalf("expected status %d for %s %s, got %d", expected, method, path, rec.Code)
	}
}

func pathHasAuthRead(path string) bool {
	return strings.HasPrefix(path, "/api/v1/audit-logs")
}

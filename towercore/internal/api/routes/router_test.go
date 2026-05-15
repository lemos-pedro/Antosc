package routes

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"towercore/internal/adapters/eltek"
	"towercore/internal/adapters/enetek"
	"towercore/internal/adapters/huawei"
	"towercore/internal/adapters/mock"
	"towercore/internal/adapters/snmp"
	"towercore/internal/api/handlers"
	"towercore/internal/core/services"
	"towercore/internal/infrastructure/config"
	"towercore/internal/infrastructure/logger"
)

const (
	testAPIKey = "test-key"
	testToken  = "test-token"
)

func testConfig() config.Config {
	return config.Config{
		AppName: "towercore-api",
		Auth: config.AuthConfig{
			APIKeyHash:      "62af8704764faf8ea82fc61ce9c4c3908b6cb97d463a634e9e587d7c885db0ef",
			BearerToken:     testToken,
			UserTokenSecret: "user-secret",
		},
		RateLimit: config.RateLimitConfig{
			WritePerMinute: 9999,
		},
	}
}

func authorizeWrite(req *http.Request) {
	req.Header.Set("X-API-Key", testAPIKey)
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("X-User-Id", "test-user")
}

func newSNMPCollectHandlerForTest() *handlers.SNMPCollectHandler {
	eventSvc := services.NewEventService(mock.NewEventRepository())
	metricSvc := services.NewMetricService(mock.NewMetricRepository())
	profiles := map[string]snmp.Profile{
		"eltek":  eltek.Profile(),
		"huawei": huawei.Profile(),
		"enetek": enetek.Profile(),
	}
	return handlers.NewSNMPCollectHandler(services.NewSNMPIngestService(metricSvc, eventSvc, profiles))
}

func newAuditHandlerForTest() *handlers.AuditHandler {
	return handlers.NewAuditHandler(services.NewAuditService(mock.NewAuditRepository()))
}

func newRouterForTest() http.Handler {
	cfg := testConfig()
	log := logger.New("info")
	towerHandler := handlers.NewTowerHandler(services.NewTowerService(mock.NewTowerRepository(), mock.NewAuditRepository()))
	eventHandler := handlers.NewEventHandler(services.NewEventService(mock.NewEventRepository()))
	metricHandler := handlers.NewMetricHandler(services.NewMetricService(mock.NewMetricRepository()))
	snmpCollectHandler := newSNMPCollectHandlerForTest()
	auditHandler := newAuditHandlerForTest()
	authRepo := mock.NewUserRepository()
	authSvc := services.NewAuthService(authRepo, cfg.Auth.UserTokenSecret, time.Hour)
	_ = authSvc.EnsureBootstrapUser(context.Background(), "admin", "admin123", "admin")
	authHandler := handlers.NewAuthHandler(authSvc)
	return NewRouter(cfg, log, towerHandler, eventHandler, metricHandler, snmpCollectHandler, auditHandler, authHandler)
}

func TestRouter_HealthRoutes(t *testing.T) {
	router := newRouterForTest()

	tests := []struct {
		name string
		path string
	}{
		{name: "v1 health", path: "/api/v1/health"},
		{name: "legacy health", path: "/health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d for %s, got %d", http.StatusOK, tt.path, rec.Code)
			}
		})
	}
}

func TestRouter_NotFound(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRouter_TowersRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/towers", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRouter_TowerDetailRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/towers/00000000-0000-0000-0000-000000000001", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRouter_TowersCreateRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/towers", bytes.NewReader([]byte(`{"name":"Tower-901","status":"online"}`)))
	authorizeWrite(req)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestRouter_EventsCreateRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader([]byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","type":"alarm","severity":"warning","message":"power fluctuation"}`)))
	authorizeWrite(req)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestRouter_EventsListRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRouter_MetricsCreateRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics", bytes.NewReader([]byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","metrics":{"voltage":48.2}}`)))
	authorizeWrite(req)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestRouter_MetricsListRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRouter_SNMPCollectRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/collect/snmp", bytes.NewReader([]byte(`{"tower_id":"00000000-0000-0000-0000-000000000001","vendor":"eltek","samples":{".1.3.6.1.4.1.12148.10.10.5.5.0":4850}}`)))
	authorizeWrite(req)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestRouter_TowersSNMPPatchRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/towers/00000000-0000-0000-0000-000000000001/snmp", bytes.NewReader([]byte(`{"vendor":"eltek","snmp_enabled":true,"snmp_target":"10.10.0.20","snmp_community":"private"}`)))
	authorizeWrite(req)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRouter_AuditLogsRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs", nil)
	authorizeWrite(req)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRouter_AuditLogByIDRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs/00000000-0000-0000-0000-000000000999", nil)
	authorizeWrite(req)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRouter_AuditLogsExportRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs/export.csv", nil)
	authorizeWrite(req)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRouter_AuthLoginRoute(t *testing.T) {
	router := newRouterForTest()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"username":"admin","password":"admin123"}`)))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

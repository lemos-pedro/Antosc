package observability

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry        *prometheus.Registry
	inFlight        prometheus.Gauge
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

func New(_ string, db *sql.DB) (*Metrics, error) {
	registry := prometheus.NewRegistry()
	m := &Metrics{
		registry: registry,
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "towercore",
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "Current number of in-flight HTTP requests.",
		}),
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "towercore",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of processed HTTP requests.",
			},
			[]string{"method", "route", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "towercore",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds.",
				Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "route", "status"},
		),
	}

	collectorsToRegister := []prometheus.Collector{
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.inFlight,
		m.requestsTotal,
		m.requestDuration,
	}
	if db != nil {
		collectorsToRegister = append(collectorsToRegister, NewDBStatsCollector(db))
	}
	for _, collector := range collectorsToRegister {
		if err := registry.Register(collector); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Instrument(route string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		m.inFlight.Inc()
		defer m.inFlight.Dec()

		next.ServeHTTP(rw, r)

		status := strconv.Itoa(rw.status)
		m.requestsTotal.WithLabelValues(r.Method, route, status).Inc()
		m.requestDuration.WithLabelValues(r.Method, route, status).Observe(time.Since(start).Seconds())
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

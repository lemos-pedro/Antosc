package observability

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

type DBStatsCollector struct {
	db *sql.DB

	maxOpenConnections *prometheus.Desc
	openConnections    *prometheus.Desc
	inUseConnections   *prometheus.Desc
	idleConnections    *prometheus.Desc
	waitCount          *prometheus.Desc
	waitDuration       *prometheus.Desc
	maxIdleClosed      *prometheus.Desc
	maxIdleTimeClosed  *prometheus.Desc
	maxLifetimeClosed  *prometheus.Desc
}

func NewDBStatsCollector(db *sql.DB) *DBStatsCollector {
	return &DBStatsCollector{
		db: db,
		maxOpenConnections: prometheus.NewDesc(
			"towercore_db_max_open_connections",
			"Configured maximum number of open DB connections.",
			nil, nil,
		),
		openConnections: prometheus.NewDesc(
			"towercore_db_open_connections",
			"Current number of established DB connections.",
			nil, nil,
		),
		inUseConnections: prometheus.NewDesc(
			"towercore_db_in_use_connections",
			"Current number of DB connections in use.",
			nil, nil,
		),
		idleConnections: prometheus.NewDesc(
			"towercore_db_idle_connections",
			"Current number of idle DB connections.",
			nil, nil,
		),
		waitCount: prometheus.NewDesc(
			"towercore_db_wait_count_total",
			"Total number of waits for a free DB connection.",
			nil, nil,
		),
		waitDuration: prometheus.NewDesc(
			"towercore_db_wait_duration_seconds_total",
			"Total time blocked waiting for a free DB connection.",
			nil, nil,
		),
		maxIdleClosed: prometheus.NewDesc(
			"towercore_db_max_idle_closed_total",
			"Total number of connections closed because of SetMaxIdleConns.",
			nil, nil,
		),
		maxIdleTimeClosed: prometheus.NewDesc(
			"towercore_db_max_idle_time_closed_total",
			"Total number of connections closed because of SetConnMaxIdleTime.",
			nil, nil,
		),
		maxLifetimeClosed: prometheus.NewDesc(
			"towercore_db_max_lifetime_closed_total",
			"Total number of connections closed because of SetConnMaxLifetime.",
			nil, nil,
		),
	}
}

func (c *DBStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.maxOpenConnections
	ch <- c.openConnections
	ch <- c.inUseConnections
	ch <- c.idleConnections
	ch <- c.waitCount
	ch <- c.waitDuration
	ch <- c.maxIdleClosed
	ch <- c.maxIdleTimeClosed
	ch <- c.maxLifetimeClosed
}

func (c *DBStatsCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()

	ch <- prometheus.MustNewConstMetric(c.maxOpenConnections, prometheus.GaugeValue, float64(stats.MaxOpenConnections))
	ch <- prometheus.MustNewConstMetric(c.openConnections, prometheus.GaugeValue, float64(stats.OpenConnections))
	ch <- prometheus.MustNewConstMetric(c.inUseConnections, prometheus.GaugeValue, float64(stats.InUse))
	ch <- prometheus.MustNewConstMetric(c.idleConnections, prometheus.GaugeValue, float64(stats.Idle))
	ch <- prometheus.MustNewConstMetric(c.waitCount, prometheus.CounterValue, float64(stats.WaitCount))
	ch <- prometheus.MustNewConstMetric(c.waitDuration, prometheus.CounterValue, stats.WaitDuration.Seconds())
	ch <- prometheus.MustNewConstMetric(c.maxIdleClosed, prometheus.CounterValue, float64(stats.MaxIdleClosed))
	ch <- prometheus.MustNewConstMetric(c.maxIdleTimeClosed, prometheus.CounterValue, float64(stats.MaxIdleTimeClosed))
	ch <- prometheus.MustNewConstMetric(c.maxLifetimeClosed, prometheus.CounterValue, float64(stats.MaxLifetimeClosed))
}

var _ prometheus.Collector = (*DBStatsCollector)(nil)

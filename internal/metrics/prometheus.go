package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	athenaConnections   *prometheus.GaugeVec
	athenaErrors        *prometheus.CounterVec
	logParserErrors     *prometheus.CounterVec
	logParserQueueSize  prometheus.Gauge
	logParserActiveJobs prometheus.Gauge
}

func NewMetrics() *Metrics {
	return &Metrics{
		athenaConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "athena_connections",
			Help: "The total number of Athena connections",
		}, []string{"dongle_id"}),
		athenaErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "athena_errors",
			Help: "The total number of Athena errors",
		}, []string{"dongle_id", "error_type"}),
		logParserErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "log_parser_errors",
			Help: "The total number of log parser errors",
		}, []string{"dongle_id", "error_type"}),
		logParserQueueSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "log_parser_queue_size",
			Help: "The current size of the log parser queue",
		}),
		logParserActiveJobs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "log_parser_active_jobs",
			Help: "The current number of active log parser jobs",
		}),
	}
}

func (m *Metrics) IncrementAthenaConnections(dongleID string) {
	m.athenaConnections.WithLabelValues(dongleID).Inc()
}

func (m *Metrics) DecrementAthenaConnections(dongleID string) {
	m.athenaConnections.WithLabelValues(dongleID).Dec()
}

func (m *Metrics) IncrementAthenaErrors(dongleID, errorType string) {
	m.athenaErrors.WithLabelValues(dongleID, errorType).Inc()
}

func (m *Metrics) IncrementLogParserErrors(dongleID, errorType string) {
	m.logParserErrors.WithLabelValues(dongleID, errorType).Inc()
}

func (m *Metrics) SetLogParserQueueSize(size float64) {
	m.logParserQueueSize.Set(size)
}

func (m *Metrics) SetLogParserActiveJobs(jobs float64) {
	m.logParserActiveJobs.Set(jobs)
}

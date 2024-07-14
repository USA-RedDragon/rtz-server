package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	athenaConnections *prometheus.GaugeVec
	athenaErrors      *prometheus.CounterVec
	logParserErrors   *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	metrics := &Metrics{
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
	}
	metrics.register()
	return metrics
}

func (m *Metrics) register() {
	prometheus.MustRegister(m.athenaConnections)
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

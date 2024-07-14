package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	athenaConnections *prometheus.GaugeVec
}

func NewMetrics() *Metrics {
	metrics := &Metrics{
		athenaConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "athena_connections",
			Help: "The total number of Athena connections",
		}, []string{"dongle_id"}),
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

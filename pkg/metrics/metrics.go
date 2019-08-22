package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	promControllerSubsystem = "controller"
)

var ClusterMetrics = &PromMetrics{}

// Instrumenter is the interface that will collect the metrics and has ability to send/expose those metrics.
type Instrumenter interface {
	SetClusterOK(namespace string, name string)
	SetClusterError(namespace string, name string)
	DeleteCluster(namespace string, name string)
}

// PromMetrics implements the instrumenter so the metrics can be managed by Prometheus.
type PromMetrics struct {
	// Metrics fields.
	clusterHealthy *prometheus.GaugeVec // clusterOk is the status of a cluster

	// Instrumentation fields.
	registry prometheus.Registerer
}

// InitPrometheusMetrics returns a init PromMetrics object.
func InitPrometheusMetrics(namespace string, registry *prometheus.Registry) {
	// Create metrics.
	clusterHealthy := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: promControllerSubsystem,
		Name:      "cluster_healthy",
		Help:      "Status of redis clusters managed by the operator.",
	}, []string{"namespace", "name"})

	ClusterMetrics.clusterHealthy = clusterHealthy
	ClusterMetrics.registry = registry

	// Register metrics on prometheus.
	ClusterMetrics.register()
}

// register will register all the required prometheus metrics on the Prometheus collector.
func (p *PromMetrics) register() {
	p.registry.MustRegister(p.clusterHealthy)
}

// SetClusterOK set the cluster status to OK
func (p *PromMetrics) SetClusterOK(namespace string, name string) {
	p.clusterHealthy.WithLabelValues(namespace, name).Set(1)
}

// SetClusterError set the cluster status to Error
func (p *PromMetrics) SetClusterError(namespace string, name string) {
	p.clusterHealthy.WithLabelValues(namespace, name).Set(0)
}

// DeleteCluster set the cluster status to Error
func (p *PromMetrics) DeleteCluster(namespace string, name string) {
	p.clusterHealthy.DeleteLabelValues(namespace, name)
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	// availabilityZonesUsedMetricKey name of the availability zones used metric.
	availabilityZonesUsedMetricKey = "nats_manager_availability_zones_used"
	// availabilityZonesUsedHelp help text for the availability zones used metric.
	availabilityZonesUsedHelp = "The number of availability zones used used by NATS Pods."

	// clusterSizeMetricKey name of the cluster size metric.
	clusterSizeMetricKey = "nats_manager_cr_cluster_size"
	// clusterSizeMetricHelp help text for the cluster size metric.
	clusterSizeMetricHelp = "The cluster size configured in the NATS CR."
)

// Perform a compile time check.
var _ Collector = &PrometheusCollector{}

//go:generate go run github.com/vektra/mockery/v2 --name=Collector --outpkg=mocks --case=underscore
type Collector interface {
	RegisterMetrics()
	RecordAvailabilityZonesUsedMetric(int)
	RecordClusterSizeMetric(int)
	ResetAvailabilityZonesUsedMetric()
	ResetClusterSizeMetric()
}

// PrometheusCollector implements the prometheus.Collector interface.
type PrometheusCollector struct {
	availabilityZonesUsed *prometheus.GaugeVec
	clusterSize           *prometheus.GaugeVec
}

// NewPrometheusCollector a new instance of Collector.
func NewPrometheusCollector() Collector {
	return &PrometheusCollector{
		availabilityZonesUsed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: availabilityZonesUsedMetricKey,
				Help: availabilityZonesUsedHelp,
			},
			nil,
		),
		clusterSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: clusterSizeMetricKey,
				Help: clusterSizeMetricHelp,
			},
			nil,
		),
	}
}

// Describe implements the prometheus.Collector interface Describe method.
func (p *PrometheusCollector) Describe(ch chan<- *prometheus.Desc) {
	p.availabilityZonesUsed.Describe(ch)
	p.clusterSize.Describe(ch)
}

// Collect implements the prometheus.Collector interface Collect method.
func (p *PrometheusCollector) Collect(ch chan<- prometheus.Metric) {
	p.availabilityZonesUsed.Collect(ch)
	p.clusterSize.Collect(ch)
}

// RegisterMetrics registers the metrics.
func (p *PrometheusCollector) RegisterMetrics() {
	metrics.Registry.MustRegister(p.availabilityZonesUsed)
	metrics.Registry.MustRegister(p.clusterSize)
}

func (p *PrometheusCollector) RecordAvailabilityZonesUsedMetric(availabilityZonesUsed int) {
	p.availabilityZonesUsed.WithLabelValues().Set(float64(availabilityZonesUsed))
}

func (p *PrometheusCollector) RecordClusterSizeMetric(clusterSize int) {
	p.clusterSize.WithLabelValues().Set(float64(clusterSize))
}

func (p *PrometheusCollector) ResetAvailabilityZonesUsedMetric() {
	p.availabilityZonesUsed.Reset()
}

func (p *PrometheusCollector) ResetClusterSizeMetric() {
	p.clusterSize.Reset()
}

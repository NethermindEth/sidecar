package prometheus

import (
	"github.com/Layr-Labs/sidecar/internal/metrics/metricsTypes"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"time"
)

type PrometheusMetricsConfig struct {
	Metrics map[metricsTypes.MetricsType][]metricsTypes.MetricsTypeConfig
}

type PrometheusMetricsClient struct {
	logger *zap.Logger
	config *PrometheusMetricsConfig

	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

func NewPrometheusMetricsClient(config *PrometheusMetricsConfig, l *zap.Logger) (*PrometheusMetricsClient, error) {
	client := &PrometheusMetricsClient{
		config: config,
		logger: l,

		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}

	client.initializeTypes()

	return client, nil
}

func (pmc *PrometheusMetricsClient) logExistingMetric(t metricsTypes.MetricsType, metric metricsTypes.MetricsTypeConfig) {
	pmc.logger.Sugar().Warnw("Prometheus metric already exists for type",
		zap.String("type", string(t)),
		zap.String("name", metric.Name),
	)
}

func (pmc *PrometheusMetricsClient) initializeTypes() {
	for t, types := range pmc.config.Metrics {
		for _, mt := range types {
			switch t {
			case metricsTypes.MetricsType_Incr:
				if _, ok := pmc.counters[mt.Name]; ok {
					pmc.logExistingMetric(t, mt)
					continue
				}
				pmc.counters[mt.Name] = prometheus.NewCounterVec(prometheus.CounterOpts{
					Name: mt.Name,
				}, mt.Labels)
				prometheus.MustRegister(pmc.counters[mt.Name])
			case metricsTypes.MetricsType_Gauge:
				if _, ok := pmc.counters[mt.Name]; ok {
					pmc.logExistingMetric(t, mt)
					continue
				}
				pmc.gauges[mt.Name] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
					Name: mt.Name,
				}, mt.Labels)
				prometheus.MustRegister(pmc.gauges[mt.Name])
			case metricsTypes.MetricsType_Timing:
				if _, ok := pmc.counters[mt.Name]; ok {
					pmc.logExistingMetric(t, mt)
					continue
				}
				pmc.histograms[mt.Name] = prometheus.NewHistogramVec(prometheus.HistogramOpts{
					Name: mt.Name,
				}, mt.Labels)
				prometheus.MustRegister(pmc.histograms[mt.Name])
			}
		}
	}
}

func (pmc *PrometheusMetricsClient) formatLabels(labels []metricsTypes.MetricsLabel) prometheus.Labels {
	l := make(prometheus.Labels)
	if labels == nil {
		return l
	}
	for _, label := range labels {
		l[label.Name] = label.Value
	}
	return l
}

func (pmc *PrometheusMetricsClient) Incr(name string, labels []metricsTypes.MetricsLabel, value float64) error {
	m, ok := pmc.counters[name]
	if !ok {
		pmc.logger.Sugar().Warnw("Prometheus incr not found",
			zap.String("name", name),
		)
		return nil
	}
	m.With(pmc.formatLabels(labels)).Add(value)
	return nil
}

func (pmc *PrometheusMetricsClient) Gauge(name string, value float64, labels []metricsTypes.MetricsLabel) error {
	m, ok := pmc.gauges[name]
	if !ok {
		pmc.logger.Sugar().Warnw("Prometheus guage not found",
			zap.String("name", name),
		)
		return nil
	}
	m.With(pmc.formatLabels(labels)).Set(value)
	return nil
}

func (pmc *PrometheusMetricsClient) Timing(name string, value time.Duration, labels []metricsTypes.MetricsLabel) error {
	return pmc.Histogram(name, value, labels)
}

func (pmc *PrometheusMetricsClient) Histogram(name string, value time.Duration, labels []metricsTypes.MetricsLabel) error {
	m, ok := pmc.histograms[name]
	if !ok {
		pmc.logger.Sugar().Warnw("Prometheus histogram not found",
			zap.String("name", name),
		)
		return nil
	}
	m.With(pmc.formatLabels(labels)).Observe(float64(value.Milliseconds()))
	return nil
}

package metricsTypes

import "time"

type IMetricsClient interface {
	Incr(name string, labels []MetricsLabel, value float64) error
	Gauge(name string, value float64, labels []MetricsLabel) error
	Timing(name string, value time.Duration, labels []MetricsLabel) error
}

type MetricsLabel struct {
	Name  string
	Value string
}

type MetricsType string

var (
	MetricsType_Incr   MetricsType = "incr"
	MetricsType_Gauge  MetricsType = "gauge"
	MetricsType_Timing MetricsType = "timing"
)

type MetricsTypeConfig struct {
	Name   string
	Labels []string
}

var (
	Metric_Incr_BlockProcessed = "blockProcessed"

	Metric_Gauge_CurrentBlockHeight = "currentBlockHeight"

	Metric_Gauge_LastDistributionRootBlockHeight = "lastDistributionRootBlockHeight"
)

var MetricTypes = map[MetricsType][]MetricsTypeConfig{
	MetricsType_Incr: {
		MetricsTypeConfig{
			Name:   Metric_Incr_BlockProcessed,
			Labels: []string{},
		},
	},
	MetricsType_Gauge: {
		MetricsTypeConfig{
			Name:   Metric_Gauge_CurrentBlockHeight,
			Labels: []string{},
		},
	},
}

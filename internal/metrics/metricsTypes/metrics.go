package metricsTypes

type IMetricsClient interface {
	Incr(name string, labels []MetricsLabel, value float64) error
	Gauge(name string, value float64, labels []MetricsLabel) error
}

type MetricsLabel struct {
	Name  string
	Value string
}

type MetricsType string

var (
	MetricsType_Incr  MetricsType = "incr"
	MetricsType_Gauge MetricsType = "gauge"
)

type MetricsTypeConfig struct {
	Name   string
	Labels []string
}

var (
	Metric_Incr_BlockProcessed = "blockProcessed"

	Metric_Gauge_CurrentBlockHeight = "currentBlockHeight"
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

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
	Metric_Incr_GrpcRequest    = "rpc.grpc.request"
	Metric_Incr_HttpRequest    = "rpc.http.request"

	Metric_Gauge_CurrentBlockHeight = "currentBlockHeight"

	Metric_Gauge_LastDistributionRootBlockHeight = "lastDistributionRootBlockHeight"

	Metric_Timing_GrpcDuration         = "rpc.grpc.duration"
	Metric_Timing_HttpDuration         = "rpc.http.duration"
	Metric_Timing_RewardsCalcDuration  = "rewards.duration"
	Metric_Timing_BlockProcessDuration = "block.process.duration"
)

var MetricTypes = map[MetricsType][]MetricsTypeConfig{
	MetricsType_Incr: {
		MetricsTypeConfig{
			Name:   Metric_Incr_BlockProcessed,
			Labels: []string{},
		},
		MetricsTypeConfig{
			Name:   Metric_Incr_GrpcRequest,
			Labels: []string{},
		},
		MetricsTypeConfig{
			Name:   Metric_Incr_HttpRequest,
			Labels: []string{},
		},
	},
	MetricsType_Gauge: {
		MetricsTypeConfig{
			Name:   Metric_Gauge_CurrentBlockHeight,
			Labels: []string{},
		},
	},
	MetricsType_Timing: {
		MetricsTypeConfig{
			Name:   Metric_Timing_GrpcDuration,
			Labels: []string{},
		},
		MetricsTypeConfig{
			Name:   Metric_Timing_HttpDuration,
			Labels: []string{},
		},
	},
}

package model

// MetricType denotes what custom type the metric is.
type MetricType string

// ReservedMetricTypes are the metric types that are reserved for the system
// due to legacy usage.
var ReservedMetricTypes = map[string]bool{
	legacyTrainingMetricsPath:   true,
	legacyValidationMetricsPath: true,
}

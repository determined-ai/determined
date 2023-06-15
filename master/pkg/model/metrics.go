package model

import "github.com/determined-ai/determined/proto/pkg/apiv1"

const (
	// ValidationMetricType designates metrics from validation runs.
	ValidationMetricType MetricType = "validation"
	// TrainingMetricType designates metrics from training runs.
	TrainingMetricType MetricType = "training"
)

// MetricType denotes what custom type the metric is.
type MetricType string

// ToString returns the string representation of the metric type.
func (t MetricType) ToString() string {
	return string(t)
}

// ToProto returns the proto representation of the metric type.
func (t MetricType) ToProto() apiv1.MetricType {
	switch t {
	case ValidationMetricType:
		return apiv1.MetricType_METRIC_TYPE_VALIDATION
	case TrainingMetricType:
		return apiv1.MetricType_METRIC_TYPE_TRAINING
	default:
		return apiv1.MetricType_METRIC_TYPE_UNSPECIFIED
	}
}

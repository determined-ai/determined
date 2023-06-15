package model

import (
	"strings"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

func (t MetricType) Validate() error {
	if len(t) == 0 {
		return status.Errorf(codes.InvalidArgument, "metric type cannot be empty")
	}
	if strings.Contains(t.ToString(), ".") {
		return status.Errorf(codes.InvalidArgument, "metric type cannot contain '.'")
	}
	return nil
}

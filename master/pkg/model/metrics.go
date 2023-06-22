package model

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/metricv1"
)

const (
	// ValidationMetricType designates metrics from validation runs.
	ValidationMetricType MetricType = "validation"
	// TrainingMetricType designates metrics from training runs.
	TrainingMetricType MetricType = "training"
)

// ValidateMetricName validates the metric name.
// CHAT: do we make a new type similar to MetricType?
func ValidateMetricName(name string) error {
	if len(name) == 0 {
		return status.Errorf(codes.InvalidArgument, "metric name cannot be empty")
	}
	return nil
}

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

// Validate validates the metric type.
func (t MetricType) Validate() error {
	if len(t) == 0 {
		return status.Errorf(codes.InvalidArgument, "metric type cannot be empty")
	}
	if strings.Contains(t.ToString(), ".") {
		return status.Errorf(codes.InvalidArgument, "metric type cannot contain '.'")
	}
	return nil
}

// MetricIdentifier packages metric type and name together.
type MetricIdentifier struct {
	Type MetricType
	Name string
}

// Serialize serializes a metric identifier to a string.
func (m MetricIdentifier) Serialize() (string, error) {
	return m.Type.ToString() + "." + m.Name, m.Type.Validate()
}

// ToProto returns the proto representation of the metric identifier.
func (m MetricIdentifier) ToProto() *metricv1.MetricName {
	return &metricv1.MetricName{
		Type: m.Type.ToString(),
		Name: m.Name,
	}
}

// DeserializeMetricIdentifier deserialize a metric identifier from a string.
func DeserializeMetricIdentifier(s string) (*MetricIdentifier, error) {
	nameAndType := strings.SplitN(s, ".", 2)
	if len(nameAndType) < 2 {
		return nil, status.Errorf(codes.InvalidArgument,
			"invalid metric identifier: '%s' expected <type>.<name>", s)
	}
	metricIDName := nameAndType[1]
	if err := ValidateMetricName(metricIDName); err != nil {
		return nil, err
	}
	metricIDType := MetricType(nameAndType[0])
	if err := metricIDType.Validate(); err != nil {
		return nil, err
	}
	return &MetricIdentifier{
		Type: metricIDType,
		Name: metricIDName,
	}, nil
}

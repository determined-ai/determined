package model

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/metricv1"
)

const (
	// ValidationMetricGroup designates metrics from validation runs.
	ValidationMetricGroup MetricGroup = "validation"
	// TrainingMetricGroup designates metrics from training runs.
	TrainingMetricGroup MetricGroup = "training"
	// InferenceMetricGroup designates metrics from inference runs.
	InferenceMetricGroup MetricGroup = "inference"
)

type metricName string

// Validate validates the metric name.
func (t metricName) Validate() error {
	if len(t) == 0 {
		return status.Errorf(codes.InvalidArgument, "metric name cannot be empty")
	}
	return nil
}

// MetricGroup denotes what custom group the metric is.
type MetricGroup string

// ToString returns the string representation of the metric group.
func (t MetricGroup) ToString() string {
	return string(t)
}

// ToProto returns the proto representation of the metric group.
func (t MetricGroup) ToProto() apiv1.MetricType {
	switch t {
	case ValidationMetricGroup:
		return apiv1.MetricType_METRIC_TYPE_VALIDATION
	case TrainingMetricGroup:
		return apiv1.MetricType_METRIC_TYPE_TRAINING
	default:
		return apiv1.MetricType_METRIC_TYPE_UNSPECIFIED
	}
}

// Validate validates the metric group.
func (t MetricGroup) Validate() error {
	if len(t) == 0 {
		return status.Errorf(codes.InvalidArgument, "metric group cannot be empty")
	}
	if strings.Contains(t.ToString(), ".") {
		return status.Errorf(codes.InvalidArgument, "metric group cannot contain '.'")
	}
	return nil
}

// MetricIdentifier packages metric group and name together.
type MetricIdentifier struct {
	Group MetricGroup
	Name  metricName
}

// ToProto returns the proto representation of the metric identifier.
func (m MetricIdentifier) ToProto() *metricv1.MetricIdentifier {
	return &metricv1.MetricIdentifier{
		Group: m.Group.ToString(),
		Name:  string(m.Name),
	}
}

// DeserializeMetricIdentifier deserialize a metric identifier from a string.
func DeserializeMetricIdentifier(s string) (*MetricIdentifier, error) {
	nameAndType := strings.SplitN(s, ".", 2)
	if len(nameAndType) < 2 {
		return nil, status.Errorf(codes.InvalidArgument,
			"invalid metric identifier: '%s' expected <group>.<name>", s)
	}
	metricIDName := metricName(nameAndType[1])
	if err := metricIDName.Validate(); err != nil {
		return nil, err
	}
	metricIDGroup := MetricGroup(nameAndType[0])
	if err := metricIDGroup.Validate(); err != nil {
		return nil, err
	}
	return &MetricIdentifier{
		Group: metricIDGroup,
		Name:  metricIDName,
	}, nil
}

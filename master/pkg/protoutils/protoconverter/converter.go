package protoconverter

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

// ProtoConverter just holds errors, allowing you to convert a large number of fields on a protobuf
// struct without checking errors until afterwards.
type ProtoConverter struct {
	err error
}

// Error returns the error collected during the conversions, if any.
func (c *ProtoConverter) Error() error {
	return c.err
}

// ToStruct converts a model.JSONObj-like type into a structpb.Struct type, remembering the error
// if any is encountered.
func (c *ProtoConverter) ToStruct(x map[string]interface{}, what string) *structpb.Struct {
	if c.err != nil {
		return nil
	}
	out, err := structpb.NewStruct(x)
	if err != nil {
		c.err = errors.Wrapf(err, "error converting %v to protobuf", what)
	}
	return out
}

// ToDoubleWrapper converts a float64 to a double wrapper.
func (c *ProtoConverter) ToDoubleWrapper(x float64) *wrapperspb.DoubleValue {
	if c.err != nil {
		return nil
	}
	return wrapperspb.Double(x)
}

// ToTimestamp converts a time.Time into a timestamppb.Timestamp.
func (c *ProtoConverter) ToTimestamp(x time.Time) *timestamppb.Timestamp {
	return timestamppb.New(x)
}

// ToCheckpointv1State converts a model.State string into a checkpointv1.State.
// TODO, rewrite, safer.
func (c *ProtoConverter) ToCheckpointv1State(state string) checkpointv1.State {
	if c.err != nil {
		return 0
	}

	out, ok := checkpointv1.State_value["STATE_"+state]
	if !ok {
		c.err = errors.Errorf("invalid checkpointv1.State: %q", state)
		return 0
	}

	return checkpointv1.State(out)
}

// ToInt32 converts an int to an int32.
func (c *ProtoConverter) ToInt32(i int) int32 {
	if c.err != nil {
		return 0
	}
	if i > math.MaxInt32 {
		c.err = errors.Errorf("int %v too big for int32", i)
		return 0
	}
	return int32(i)
}

// ToUUID converts a string to a uuid.UUID.
func (c *ProtoConverter) ToUUID(x string) uuid.UUID {
	if c.err != nil {
		return uuid.UUID{}
	}

	y, err := uuid.Parse(x)
	if err != nil {
		c.err = fmt.Errorf("string %s is not a valid uuid", x)
		return uuid.UUID{}
	}

	return y
}

// ToUUIDList converts a list of strings to a list of uuid.UUIDs.
func (c *ProtoConverter) ToUUIDList(x []string) []uuid.UUID {
	var uuidList []uuid.UUID
	if c.err != nil {
		return uuidList
	}

	if len(x) == 0 {
		return uuidList
	}

	for _, cStr := range x {
		cUUID, err := uuid.Parse(cStr)
		if err != nil {
			c.err = fmt.Errorf("string %s is not a valid uuid", cStr)
			return uuidList
		}
		uuidList = append(uuidList, cUUID)
	}

	return uuidList
}

// ToStringList converts a list of uuid.UUIDs to list of strings.
func (c *ProtoConverter) ToStringList(x []uuid.UUID) []string {
	var strUUIDList []string
	if c.err != nil {
		return strUUIDList
	}

	if len(x) == 0 {
		return strUUIDList
	}

	for _, cUUID := range x {
		cStr := cUUID.String()
		if cStr == "" {
			c.err = fmt.Errorf("uuid %v is not a valid uuid", cUUID)
			return strUUIDList
		}
		strUUIDList = append(strUUIDList, cStr)
	}

	return strUUIDList
}

// ToCheckpointState converts a proto chechkpoint state internal state representation.
func (c *ProtoConverter) ToCheckpointState(x checkpointv1.State) model.State {
	if c.err != nil {
		return ""
	}

	switch x {
	case checkpointv1.State_STATE_UNSPECIFIED:
		return ""
	case checkpointv1.State_STATE_ACTIVE:
		return model.ActiveState
	case checkpointv1.State_STATE_COMPLETED:
		return model.CompletedState
	case checkpointv1.State_STATE_DELETED:
		return model.DeletedState
	case checkpointv1.State_STATE_ERROR:
		return model.ErrorState
	default:
		c.err = fmt.Errorf("state %s is not a valid state to the backend", x)
		return ""
	}
}

// ToMetricGroup converts a proto metric group to internal metric group representation.
func (c *ProtoConverter) ToMetricGroup(x apiv1.MetricType) model.MetricGroup {
	if c.err != nil {
		return ""
	}

	switch x {
	case apiv1.MetricType_METRIC_TYPE_UNSPECIFIED:
		return ""
	case apiv1.MetricType_METRIC_TYPE_TRAINING:
		return model.TrainingMetricGroup
	case apiv1.MetricType_METRIC_TYPE_VALIDATION:
		return model.ValidationMetricGroup
	default:
		c.err = fmt.Errorf("metric group %s is not a valid metric group to the backend", x)
		return ""
	}
}

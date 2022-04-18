package protoconverter

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
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

// Convert JSON data from db into Note objects.
func (c *ProtoConverter) ToProjectNotes(j []model.JSONObj) ([]*projectv1.Note, error) {
	var notes []*projectv1.Note
	b, err := json.Marshal(j)
	if err != nil {
		return notes, err
	}
	err = json.Unmarshal(b, &notes)
	return notes, err
}

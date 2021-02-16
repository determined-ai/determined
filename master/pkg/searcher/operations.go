package searcher

import (
	"encoding/json"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

// Operation represents the base interface for possible operations that a search method can return.
type Operation interface{}

type (
	// OperationType encodes the underlying type of an Operation for serialization.
	OperationType int

	// OperationWithType is an operation with a serializable repr of its underlying type.
	OperationWithType struct {
		OperationType
		Operation
	}

	// OperationList is []Operation that handles marshaling and unmarshaling heterogeneous
	// operations to and from their correct underlying types.
	OperationList []Operation
)

// All the operation types that support serialization.
const (
	CreateOperation OperationType = iota
	TrainOperation
	ValidateOperation
	CloseOperation
)

// MarshalJSON implements json.Marshaler.
func (l OperationList) MarshalJSON() ([]byte, error) {
	var typedOps []OperationWithType
	for _, op := range l {
		typedOp := OperationWithType{Operation: op}
		switch op.(type) {
		case Create:
			typedOp.OperationType = CreateOperation
		case Train:
			typedOp.OperationType = TrainOperation
		case Validate:
			typedOp.OperationType = ValidateOperation
		case Close:
			typedOp.OperationType = CloseOperation
		default:
			return nil, fmt.Errorf("unable to serialize %T as operation", op)
		}
		typedOps = append(typedOps, typedOp)
	}
	return json.Marshal(typedOps)
}

// UnmarshalJSON implements json.Unmarshaler.
func (l *OperationList) UnmarshalJSON(b []byte) error {
	var typedOps []OperationWithType
	if err := json.Unmarshal(b, &typedOps); err != nil {
		return err
	}
	var ops OperationList
	for _, typedOp := range typedOps {
		b, err := json.Marshal(typedOp.Operation)
		if err != nil {
			return err
		}
		switch typedOp.OperationType {
		case CreateOperation:
			var op Create
			if err := json.Unmarshal(b, &op); err != nil {
				return err
			}
			ops = append(ops, op)
		case TrainOperation:
			var op Train
			if err := json.Unmarshal(b, &op); err != nil {
				return err
			}
			ops = append(ops, op)
		case ValidateOperation:
			var op Validate
			if err := json.Unmarshal(b, &op); err != nil {
				return err
			}
			ops = append(ops, op)
		case CloseOperation:
			var op Close
			if err := json.Unmarshal(b, &op); err != nil {
				return err
			}
			ops = append(ops, op)
		default:
			return fmt.Errorf("unable to deserialize %d as operation", typedOp.OperationType)
		}
	}
	*l = ops
	return nil
}

// Requested is a convenience interface for operations that were requested by a searcher method
// for a specific trial.
type Requested interface {
	GetRequestID() model.RequestID
}

// Runnable represents any runnable operation. It acts as a sum type for Train, Validate,
// Checkpoints and any future operations that the harness may run.
type Runnable interface {
	Requested
	Runnable()
}

// Create a new trial for the search method.
type Create struct {
	RequestID model.RequestID `json:"request_id"`
	// TrialSeed must be a value between 0 and 2**31 - 1.
	TrialSeed             uint32                      `json:"trial_seed"`
	Hparams               hparamSample                `json:"hparams"`
	Checkpoint            *Checkpoint                 `json:"checkpoint"`
	WorkloadSequencerType model.WorkloadSequencerType `json:"workload_sequencer_type"`
}

// NewCreate initializes a new Create operation with a new request ID and the given hyperparameters.
func NewCreate(
	rand *nprand.State, s hparamSample, sequencerType model.WorkloadSequencerType) Create {
	return Create{
		RequestID:             model.NewRequestID(rand),
		TrialSeed:             uint32(rand.Int64n(1 << 31)),
		Hparams:               s,
		WorkloadSequencerType: sequencerType,
	}
}

// NewCreateFromParent initializes a new Create operation with a new request ID and the given
// hyperparameters and checkpoint to initially load from.
func NewCreateFromParent(
	rand *nprand.State, s hparamSample, parentID model.RequestID,
	sequencerType model.WorkloadSequencerType,
) Create {
	create := NewCreate(rand, s, sequencerType)
	create.Checkpoint = &Checkpoint{parentID}
	return create
}

func (create Create) String() string {
	if create.Checkpoint == nil {
		return fmt.Sprintf("{Create %s, seed %d}", create.RequestID, create.TrialSeed)
	}
	return fmt.Sprintf(
		"{Create %s, seed %d, parent %v}", create.RequestID, create.TrialSeed,
		create.Checkpoint.RequestID,
	)
}

// GetRequestID implemented Requested.
func (create Create) GetRequestID() model.RequestID { return create.RequestID }

// Checkpoint indicates which trial the trial created by a Create should inherit from.
type Checkpoint struct {
	RequestID model.RequestID
}

func (c Checkpoint) String() string {
	return fmt.Sprintf("{Checkpoint %s}", c.RequestID)
}

// Train is an operation emitted by search methods to signal the trial train for a specified length.
type Train struct {
	RequestID model.RequestID
	Length    model.Length
}

// NewTrain returns a new train operation.
func NewTrain(requestID model.RequestID, length model.Length) Train {
	return Train{requestID, length}
}

func (t Train) String() string {
	return fmt.Sprintf("{Train %s, %s}", t.RequestID, t.Length)
}

// Runnable implements Runnable.
func (t Train) Runnable() {}

// GetRequestID implemented Requested.
func (t Train) GetRequestID() model.RequestID { return t.RequestID }

// Validate is an operation emitted by search methods to signal the trial to validate.
type Validate struct {
	RequestID model.RequestID
}

// NewValidate returns a new validate operation.
func NewValidate(requestID model.RequestID) Validate {
	return Validate{requestID}
}

func (v Validate) String() string {
	return fmt.Sprintf("{Validate %s}", v.RequestID)
}

// Runnable implements Runnable.
func (v Validate) Runnable() {}

// GetRequestID implemented Requested.
func (v Validate) GetRequestID() model.RequestID { return v.RequestID }

// Close the trial with the given trial id.
type Close struct {
	RequestID model.RequestID `json:"request_id"`
}

// NewClose initializes a new Close operation for the request ID.
func NewClose(requestID model.RequestID) Close {
	return Close{
		RequestID: requestID,
	}
}

func (close Close) String() string {
	return fmt.Sprintf("{Close %s}", close.RequestID)
}

// GetRequestID implemented Requested.
func (close Close) GetRequestID() model.RequestID { return close.RequestID }

// Shutdown marks the searcher as completed.
type Shutdown struct {
	Failure bool
}

// NewShutdown initializes a Shutdown operation for the searcher.
func NewShutdown() Shutdown {
	return Shutdown{}
}

func (shutdown Shutdown) String() string {
	return "{Shutdown}"
}

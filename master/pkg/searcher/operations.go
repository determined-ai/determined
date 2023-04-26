package searcher

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/proto/pkg/experimentv1"

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
	CreateOperation              OperationType = 0
	TrainOperation               OperationType = 1
	ValidateOperation            OperationType = 2
	CloseOperation               OperationType = 4
	ValidateAfterOperation       OperationType = 5
	SetSearcherProgressOperation OperationType = 6
)

// MarshalJSON implements json.Marshaler.
func (l OperationList) MarshalJSON() ([]byte, error) {
	var typedOps []OperationWithType
	for _, op := range l {
		typedOp := OperationWithType{Operation: op}
		switch op.(type) {
		case Create:
			typedOp.OperationType = CreateOperation
		case ValidateAfter:
			typedOp.OperationType = ValidateAfterOperation
		case Close:
			typedOp.OperationType = CloseOperation
		case SetSearcherProgress:
			typedOp.OperationType = SetSearcherProgressOperation
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
		case ValidateAfterOperation:
			var op ValidateAfter
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

// Create a new trial for the search method.
type Create struct {
	RequestID model.RequestID `json:"request_id"`
	// TrialSeed must be a value between 0 and 2**31 - 1.
	TrialSeed             uint32                      `json:"trial_seed"`
	Hparams               HParamSample                `json:"hparams"`
	Checkpoint            *Checkpoint                 `json:"checkpoint"`
	WorkloadSequencerType model.WorkloadSequencerType `json:"workload_sequencer_type"`
}

// NewCreate initializes a new Create operation with a new request ID and the given hyperparameters.
func NewCreate(
	rand *nprand.State, s HParamSample, sequencerType model.WorkloadSequencerType,
) Create {
	return Create{
		RequestID:             model.NewRequestID(rand),
		TrialSeed:             uint32(rand.Int64n(1 << 31)),
		Hparams:               s,
		WorkloadSequencerType: sequencerType,
	}
}

// NewCreateFromCheckpoint initializes a new Create operation with a new request ID and the given
// hyperparameters and checkpoint to initially load from.
func NewCreateFromCheckpoint(
	rand *nprand.State, s HParamSample, parentID model.RequestID,
	sequencerType model.WorkloadSequencerType,
) Create {
	create := NewCreate(rand, s, sequencerType)
	create.Checkpoint = &Checkpoint{parentID}
	return create
}

// CreateFromProto initializes a new Create operation from an
// experimentv1.SearcherOperation_CreateTrial.
func CreateFromProto(
	protoSearcherOp *experimentv1.SearcherOperation_CreateTrial,
	sequencerType model.WorkloadSequencerType,
) (*Create, error) {
	requestID, err := uuid.Parse(protoSearcherOp.CreateTrial.RequestId)
	if err != nil {
		return nil, fmt.Errorf("unparseable trial ID %s", protoSearcherOp.CreateTrial.RequestId)
	}
	// TODO: Determine whether trial seed is set on client or on master.
	trialSeed := uint32(42)
	var hparams HParamSample
	if err = json.Unmarshal([]byte(protoSearcherOp.CreateTrial.Hyperparams), &hparams); err != nil {
		// TODO: Should we return this err instead?
		return nil, fmt.Errorf("unparseable hyperparams %s", protoSearcherOp.CreateTrial.Hyperparams)
	}
	return &Create{
		RequestID:             model.RequestID(requestID),
		TrialSeed:             trialSeed,
		Hparams:               hparams,
		WorkloadSequencerType: sequencerType,
	}, nil
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

// ValidateAfter is an operation emitted by search methods to signal the trial train until
// its total batches trained equals the specified length.
type ValidateAfter struct {
	RequestID model.RequestID
	Length    uint64
}

// NewValidateAfter returns a new train operation.
func NewValidateAfter(requestID model.RequestID, length uint64) ValidateAfter {
	return ValidateAfter{requestID, length}
}

// ValidateAfterFromProto creates a ValidateAfter operation from its protobuf representation.
func ValidateAfterFromProto(
	op *experimentv1.TrialOperation_ValidateAfter,
) (*ValidateAfter, error) {
	requestID, err := uuid.Parse(op.ValidateAfter.RequestId)
	if err != nil {
		return nil, fmt.Errorf("unparseable trial ID %s", op.ValidateAfter.RequestId)
	}
	return &ValidateAfter{
		RequestID: model.RequestID(requestID),
		Length:    op.ValidateAfter.Length,
	}, nil
}

func (t ValidateAfter) String() string {
	return fmt.Sprintf("{ValidateAfter %s, %v}", t.RequestID, t.Length)
}

// GetRequestID implemented Requested.
func (t ValidateAfter) GetRequestID() model.RequestID { return t.RequestID }

// ToProto converts a searcher.ValidateAfter to its protobuf representation.
func (t ValidateAfter) ToProto() *experimentv1.ValidateAfterOperation {
	return &experimentv1.ValidateAfterOperation{Length: t.Length}
}

// SetSearcherProgress sets the progress of the custom searcher.
type SetSearcherProgress struct {
	Progress float64
}

// SetSearcherProgressFromProto creates a SetSearcherProgress from its protobuf representation.
func SetSearcherProgressFromProto(
	op *experimentv1.SearcherOperation_SetSearcherProgress,
) SetSearcherProgress {
	return SetSearcherProgress{Progress: op.SetSearcherProgress.Progress}
}

// Close the trial with the given trial ID.
type Close struct {
	RequestID model.RequestID `json:"request_id"`
}

// NewClose initializes a new Close operation for the request ID.
func NewClose(requestID model.RequestID) Close {
	return Close{
		RequestID: requestID,
	}
}

// CloseFromProto returns a Close operation from its protobuf representation.
func CloseFromProto(
	op *experimentv1.SearcherOperation_CloseTrial,
) (*Close, error) {
	requestID, err := uuid.Parse(op.CloseTrial.RequestId)
	if err != nil {
		return nil, fmt.Errorf("unparseable trial ID %s", op.CloseTrial.RequestId)
	}
	return &Close{
		RequestID: model.RequestID(requestID),
	}, nil
}

func (close Close) String() string {
	return fmt.Sprintf("{Close %s}", close.RequestID)
}

// GetRequestID implemented Requested.
func (close Close) GetRequestID() model.RequestID { return close.RequestID }

// Shutdown marks the searcher as completed.
type Shutdown struct {
	Cancel  bool
	Failure bool
}

// NewShutdown initializes a Shutdown operation for the searcher.
func NewShutdown() Shutdown {
	return Shutdown{}
}

// ShutdownFromProto creates a Shutdown from its protobuf representation.
func ShutdownFromProto(
	op *experimentv1.SearcherOperation_ShutDown,
) (*Shutdown, error) {
	return &Shutdown{
		Cancel:  op.ShutDown.Cancel,
		Failure: op.ShutDown.Failure,
	}, nil
}

func (shutdown Shutdown) String() string {
	return fmt.Sprintf("{Shutdown Cancel: %v Failure: %v}", shutdown.Cancel, shutdown.Failure)
}

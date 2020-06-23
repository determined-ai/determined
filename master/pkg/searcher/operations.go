package searcher

import (
	"bytes"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

// Operation represents the base interface for possible operations that a search method can return.
type Operation interface{}

// RequestID links all operations with the same ID to a single trial create request.
type RequestID uuid.UUID

func newRequestID(r io.Reader) RequestID {
	var u uuid.UUID
	if _, err := io.ReadFull(r, u[:]); err != nil {
		// We always read from an `nprand.State`, which should
		// not return an error in practice.
		panic(fmt.Sprintf("unexpected error creating request ID: %v", err))
	}

	// Ensure that the underlying UUID is a valid UUIDv4.
	u[6] = (u[6] & 0x0f) | 0x40 // Version 4.
	u[8] = (u[8] & 0x3f) | 0x80 // Variant is 10.
	return RequestID(u)
}

// MarshalText returns the marshaled form of this ID, which is the string form of the underlying
// UUID.
func (r RequestID) MarshalText() ([]byte, error) {
	return []byte(uuid.UUID(r).String()), nil
}

// UnmarshalText unmarshals this ID from a text representation.
func (r *RequestID) UnmarshalText(data []byte) error {
	u, err := uuid.ParseBytes(data)
	if err != nil {
		return err
	}
	*r = RequestID(u)
	return nil
}

// Before determines whether this UUID is strictly lexicographically less (comparing the sequences
// of bytes) than another one.
func (r RequestID) Before(s RequestID) bool {
	return bytes.Compare(r[:], s[:]) == -1
}

func (r RequestID) String() string {
	return uuid.UUID(r).String()
}

// Parse decodes s into a request id or returns an error.
func Parse(s string) (RequestID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return RequestID{}, err
	}
	return RequestID(parsed), nil
}

// MustParse decodes s into a request id or panics.
func MustParse(s string) RequestID {
	parsed, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return parsed
}

// Create a new trial for the search method.
type Create struct {
	RequestID RequestID `json:"request_id"`
	// TrialSeed must be a value between 0 and 2**31 - 1.
	TrialSeed             uint32                      `json:"trial_seed"`
	Hparams               hparamSample                `json:"hparams"`
	Checkpoint            *WorkloadOperation          `json:"checkpoint"`
	WorkloadSequencerType model.WorkloadSequencerType `json:"workload_sequencer_type"`
}

// NewCreate initializes a new Create operation with a new request ID and the given hyperparameters.
func NewCreate(
	rand *nprand.State, s hparamSample, sequencerType model.WorkloadSequencerType) Create {
	return Create{
		RequestID:             newRequestID(rand),
		TrialSeed:             uint32(rand.Int64n(1 << 31)),
		Hparams:               s,
		WorkloadSequencerType: sequencerType,
	}
}

// NewCreateFromCheckpoint initializes a new Create operation with a new request ID and the given
// hyperparameters and checkpoint to initially load from.
func NewCreateFromCheckpoint(
	rand *nprand.State, s hparamSample, ckptRequestID RequestID, ckptStepID int,
	sequencerType model.WorkloadSequencerType,
) Create {
	create := NewCreate(rand, s, sequencerType)
	checkpoint := NewCheckpoint(ckptRequestID, ckptStepID)
	create.Checkpoint = &checkpoint
	return create
}

func (create Create) String() string {
	if create.Checkpoint == nil {
		return fmt.Sprintf("{Create %s, seed %d}", create.RequestID, create.TrialSeed)
	}
	return fmt.Sprintf(
		"{Create %s, seed %d, checkpoint %v}", create.RequestID, create.TrialSeed, create.Checkpoint,
	)
}

// WorkloadOperation encompasses the intent for a searcher to run a workload on a trial.
type WorkloadOperation struct {
	RequestID  RequestID `json:"request_id"`
	Kind       Kind      `json:"kind"`
	StepID     int       `json:"step_id"`
	NumBatches int       `json:"num_batches"`
}

// NewTrain signals to a trial runner that it should run a training step.
func NewTrain(requestID RequestID, stepID, numBatches int) WorkloadOperation {
	return WorkloadOperation{
		RequestID:  requestID,
		Kind:       RunStep,
		StepID:     stepID,
		NumBatches: numBatches,
	}
}

// NewCheckpoint signals to the trial runner that the current model state should be checkpointed.
func NewCheckpoint(requestID RequestID, stepID int) WorkloadOperation {
	return WorkloadOperation{
		RequestID: requestID,
		Kind:      CheckpointModel,
		StepID:    stepID,
	}
}

// NewValidate signals to a trial runner it should compute validation metrics.
func NewValidate(requestID RequestID, stepID int) WorkloadOperation {
	return WorkloadOperation{
		RequestID: requestID,
		Kind:      ComputeValidationMetrics,
		StepID:    stepID,
	}
}

func (wo WorkloadOperation) String() string {
	return fmt.Sprintf("{Workload %s %s, step %d}", wo.Kind, wo.RequestID, wo.StepID)
}

// Close the trial with the given trial id.
type Close struct {
	RequestID RequestID `json:"request_id"`
}

// NewClose initializes a new Close operation for the request ID.
func NewClose(requestID RequestID) Close {
	return Close{
		RequestID: requestID,
	}
}

func (close Close) String() string {
	return fmt.Sprintf("{Close %s}", close.RequestID)
}

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

package searcher

import (
	"bytes"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
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

// Requested is a convenience interface for operations that were requested by a searcher method
// for a specific trial.
type Requested interface {
	GetRequestID() RequestID
}

// Runnable represents any runnable operation. It acts as a sum type for Train, Validate,
// Checkpoints and any future operations that the harness may run.
type Runnable interface {
	Requested
	Runnable()
}

// Create a new trial for the search method.
type Create struct {
	RequestID RequestID `json:"request_id"`
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
		RequestID:             newRequestID(rand),
		TrialSeed:             uint32(rand.Int64n(1 << 31)),
		Hparams:               s,
		WorkloadSequencerType: sequencerType,
	}
}

// NewCreateFromCheckpoint initializes a new Create operation with a new request ID and the given
// hyperparameters and checkpoint to initially load from.
func NewCreateFromCheckpoint(
	rand *nprand.State, s hparamSample, checkpoint Checkpoint,
	sequencerType model.WorkloadSequencerType,
) Create {
	create := NewCreate(rand, s, sequencerType)
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

// GetRequestID implemented Requested.
func (create Create) GetRequestID() RequestID { return create.RequestID }

// Train is an operation emitted by search methods to signal the trial train for a specified length.
type Train struct {
	RequestID RequestID
	Length    expconf.Length
}

// NewTrain returns a new train operation.
func NewTrain(requestID RequestID, length expconf.Length) Train {
	return Train{requestID, length}
}

func (t Train) String() string {
	return fmt.Sprintf("{Train %s, %s}", t.RequestID, t.Length)
}

// Runnable implements Runnable.
func (t Train) Runnable() {}

// GetRequestID implemented Requested.
func (t Train) GetRequestID() RequestID { return t.RequestID }

// Validate is an operation emitted by search methods to signal the trial to validate.
type Validate struct {
	RequestID RequestID
}

// NewValidate returns a new validate operation.
func NewValidate(requestID RequestID) Validate {
	return Validate{requestID}
}

func (v Validate) String() string {
	return fmt.Sprintf("{Validate %s}", v.RequestID)
}

// Runnable implements Runnable.
func (v Validate) Runnable() {}

// GetRequestID implemented Requested.
func (v Validate) GetRequestID() RequestID { return v.RequestID }

// Checkpoint is an operation emitted by search methods to signal the trial to checkpoint.
type Checkpoint struct {
	RequestID RequestID
}

// NewCheckpoint returns a new checkpoint operation.
func NewCheckpoint(requestID RequestID) Checkpoint {
	return Checkpoint{requestID}
}

func (c Checkpoint) String() string {
	return fmt.Sprintf("{Checkpoint %s}", c.RequestID)
}

// Runnable implements Runnable.
func (c Checkpoint) Runnable() {}

// GetRequestID implemented Requested.
func (c Checkpoint) GetRequestID() RequestID { return c.RequestID }

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

// GetRequestID implemented Requested.
func (close Close) GetRequestID() RequestID { return close.RequestID }

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

package searcher

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/pkg/nprand"
)

// Action is an action that a searcher would like to perform.
type Action interface {
	searcherAction()
}

// Create is a directive from the searcher to create a new run.
type Create struct {
	RequestID model.RequestID `json:"request_id"`
	// TrialSeed must be a value between 0 and 2**31 - 1.
	TrialSeed uint32       `json:"trial_seed"`
	Hparams   HParamSample `json:"hparams"`
}

// searcherAction (Create) implements SearcherAction.
func (Create) searcherAction() {}

func (action Create) String() string {
	return fmt.Sprintf(
		"Create{TrialSeed: %d, Hparams: %v, RequestID: %d}",
		action.TrialSeed, action.Hparams, action.RequestID,
	)
}

// NewCreate initializes a new Create operation with a new request ID and the given hyperparameters.
func NewCreate(
	rand *nprand.State, s HParamSample,
) Create {
	return Create{
		RequestID: model.NewRequestID(rand),
		TrialSeed: uint32(rand.Int64n(1 << 31)),
		Hparams:   s,
	}
}

// Stop is a directive from the searcher to stop a run.
type Stop struct {
	RequestID model.RequestID `json:"request_id"`
}

// SearcherAction (Stop) implements SearcherAction.
func (Stop) searcherAction() {}

// NewStop initializes a new Stop action with the given Run ID.
func NewStop(requestID model.RequestID) Stop {
	return Stop{RequestID: requestID}
}

func (action Stop) String() string {
	return fmt.Sprintf("Stop{RequestID: %d}", action.RequestID)
}

// Shutdown marks the searcher as completed.
type Shutdown struct {
	Cancel  bool
	Failure bool
}

// SearcherAction (Shutdown) implements SearcherAction.
func (Shutdown) searcherAction() {}

func (shutdown Shutdown) String() string {
	return fmt.Sprintf("{Shutdown Cancel: %v Failure: %v}", shutdown.Cancel, shutdown.Failure)
}

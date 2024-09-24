package searcher

import (
	"fmt"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

type Action interface {
	String() string
}

type Create struct {
	// RunSeed must be a value between 0 and 2**31 - 1.
	RunSeed uint32       `json:"run_seed"`
	Hparams HParamSample `json:"hparams"`
	// This is only used for adaptive ASHA to associate runs created with subsearches.
	SubSearchID int `json:"sub_search_id"`
}

type Stop struct {
	RunID int32 `json:"run_id"`
}

func (action Create) String() string {
	return fmt.Sprintf("Create{RunSeed: %d, Hparams: %v, SubSearchID: %d}", action.RunSeed, action.Hparams, action.SubSearchID)
}

// NewCreate initializes a new Create action with the given random state and hyperparameters.
func NewCreate(
	rand *nprand.State, s HParamSample,
) Create {
	return Create{
		RunSeed: uint32(rand.Int64n(1 << 31)),
		Hparams: s,
	}
}

func NewStop(runID int32) Stop {
	return Stop{RunID: runID}
}

func (action Stop) String() string {
	return fmt.Sprintf("Stop{RunID: %d}", action.RunID)
}

// Shutdown marks the searcher as completed.
type Shutdown struct {
	Cancel  bool
	Failure bool
}

func (shutdown Shutdown) String() string {
	return fmt.Sprintf("{Shutdown Cancel: %v Failure: %v}", shutdown.Cancel, shutdown.Failure)
}

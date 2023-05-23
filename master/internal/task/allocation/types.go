package allocation

import (
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

type (
	// MarkResourcesDaemon marks the given reservation as a daemon. In the event of a normal exit,
	// the allocation will not wait for it to exit on its own and instead will kill it and instead
	// await it's hopefully quick termination.
	MarkResourcesDaemon struct {
		ResourcesID sproto.ResourcesID
	}
	// AllocationExited summarizes the exit status of an allocation.
	AllocationExited struct {
		// userRequestedStop is when a container unexpectedly exits with 0.
		UserRequestedStop bool
		Err               error
		FinalState        AllocationState
	}
	// BuildTaskSpec is a message to request the task spec from the parent task. This
	// is just a hack since building a task spec cant be semi-costly and we want to defer it
	// until it is needed (we save stuff to the DB and make SSH keys, doing this for 10k trials
	// at once is real bad.
	BuildTaskSpec struct{}
	// AllocationState requests allocation state. A copy is filled and returned.
	AllocationState struct {
		State     model.AllocationState
		Resources map[sproto.ResourcesID]sproto.ResourcesSummary
		Ready     bool

		Addresses  map[sproto.ResourcesID][]cproto.Address
		Containers map[sproto.ResourcesID][]cproto.Container
	}
	// SetAllocationProxyAddress manually sets the allocation proxy address.
	SetAllocationProxyAddress struct {
		ProxyAddress string
	}
	// IsAllocationRestoring asks the allocation if it is in the middle of a restore.
	IsAllocationRestoring struct{}
)

package allocationservice

import (
	"github.com/determined-ai/determined/master/internal/task/tproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

// DefaultService is the default AllocationService.
var DefaultService AllocationService

// SetDefaultService sets the default, singleton AllocationService.
func SetDefaultService(s AllocationService) {
	DefaultService = s
}

// AllocationService is a service that just porvides allocation state. It just exists
// to prevent an import cycle between internal/rm/agentrm and internal/task from the
// old `GetContainerResourcesState` that recovers container state on restart. We shouldn't
// expand this unless it is useful for mocking in tests. Ideally, it is removed.
type AllocationService interface {
	State(id model.AllocationID) (tproto.AllocationState, error)
}

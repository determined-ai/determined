package rm

import (
	"context"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

// ResourceManagerAuthZRBAC is RBAC authorization for resource managers.
type ResourceManagerAuthZRBAC struct{}

// FilterResourcePools takes in a slice of all resource pools and the IDs of the workspaces
// the curUser has access to. It outputs a list of resource pools the user has access to by
// combining all unbound pools with those bound to workspaces the user has access to.
func (r *ResourceManagerAuthZRBAC) FilterResourcePools(
	ctx context.Context, curUser model.User, resourcePools []*resourcepoolv1.ResourcePool,
	accessibleWorkspaces []int32,
) ([]*resourcepoolv1.ResourcePool, error) {
	availablePools := set.New[string]()

	// Start with all the bound RPs they have access to
	availableWorkspaceSet := set.FromSlice(accessibleWorkspaces)
	allBindings, err := db.GetAllBindings(ctx)
	if err != nil {
		return nil, err
	}
	for _, binding := range allBindings {
		if availableWorkspaceSet.Contains(int32(binding.WorkspaceID)) {
			availablePools.Insert(binding.PoolName)
		}
	}

	// Now add unbound pools
	var allPoolNames []string
	for _, rp := range resourcePools {
		allPoolNames = append(allPoolNames, rp.Name)
	}
	unboundPools, err := db.GetUnboundRPs(ctx, allPoolNames)
	if err != nil {
		return nil, err
	}
	for _, poolName := range unboundPools {
		availablePools.Insert(poolName)
	}

	// Now we can filter using our set
	var filteredPools []*resourcepoolv1.ResourcePool
	for _, resourcePool := range resourcePools {
		if availablePools.Contains(resourcePool.Name) {
			filteredPools = append(filteredPools, resourcePool)
		}
	}

	return filteredPools, nil
}

func init() {
	AuthZProvider.Register("rbac", &ResourceManagerAuthZRBAC{})
}

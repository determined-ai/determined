package resourcepoolrbac

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

// ResourcePoolAuthZBasic is classic OSS Determined authentication for workspaces.
type ResourcePoolAuthZBasic struct{}

// FilterResourcePools always returns provided list and a nil error.
func (a *ResourcePoolAuthZBasic) FilterResourcePools(
	ctx context.Context, curUser model.User, resourcePools []*resourcepoolv1.ResourcePool,
) ([]*resourcepoolv1.ResourcePool, error) {
	return resourcePools, nil
}

func init() {
	AuthZProvider.Register("basic", &ResourcePoolAuthZBasic{})
}

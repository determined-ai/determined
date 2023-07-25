package rm

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

// ResourceManagerAuthZ is the interface for resource manager authorization.
type ResourceManagerAuthZ interface {
	// GET /api/v1/resource-pools
	FilterResourcePools(
		ctx context.Context, curUser model.User, resourcePools []*resourcepoolv1.ResourcePool,
		accessibleWorkspaces []int32,
	) ([]*resourcepoolv1.ResourcePool, error)
}

// AuthZProvider provides ResourceManagerAuthZ implementations.
var AuthZProvider authz.AuthZProviderType[ResourceManagerAuthZ]

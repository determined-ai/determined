package resourcepoolrbac

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

// ResourcePoolAuthZ is the interface for resource pool authorization.
type ResourcePoolAuthZ interface {
	// GET /api/v1/resource-pools
	FilterResourcePools(
		ctx context.Context, curUser model.User, resourcePools []*resourcepoolv1.ResourcePool,
	) ([]*resourcepoolv1.ResourcePool, error)
}

// AuthZProvider providers ResourcePoolAuthZ implementations.
var AuthZProvider authz.AuthZProviderType[ResourcePoolAuthZ]

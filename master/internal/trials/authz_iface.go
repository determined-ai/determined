package trials

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialAuthZ describes authz methods for experiments.
type TrialAuthZ interface {
	// POST /trial-comparison/collections
	CanCreateTrialCollection(
		ctx context.Context, curUser *model.User, projectID int32,
	) (canGetCollections bool, serverError error)

	// POST /trial-comparison/query
	// POST /trial-comparison/update-trial-tags
	AuthFilterTrialsQuery(
		ctx context.Context,
		curUser *model.User,
		query *bun.SelectQuery,
		update bool,
	) (*bun.SelectQuery, error)

	// GET /trial-comparison/collections
	AuthFilterCollectionsReadQuery(
		ctx context.Context,
		curUser *model.User,
		query *bun.SelectQuery,
	) (*bun.SelectQuery, error)

	// PATCH /trial-comparison/collections
	AuthFilterCollectionsUpdateQuery(
		ctx context.Context,
		curUser *model.User,
		query *bun.UpdateQuery,
	) (*bun.UpdateQuery, error)

	// DELETE /trial-comparison/collections
	AuthFilterCollectionsDeleteQuery(
		ctx context.Context,
		curUser *model.User,
		query *bun.DeleteQuery,
	) (*bun.DeleteQuery, error)
}

// AuthZProvider is the authz registry for experiments.
var AuthZProvider authz.AuthZProviderType[TrialAuthZ]

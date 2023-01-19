package trials

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialAuthZBasic is basic OSS Determined authentication.
type TrialAuthZBasic struct{}

// GET /trial-comparison/collections
func (a *TrialAuthZBasic) AuthFilterCollectionsReadQuery(
	ctx context.Context,
	curUser *model.User,
	query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	return query, nil
}

// PATCH /trial-comparison/collections
func (a *TrialAuthZBasic) AuthFilterCollectionsUpdateQuery(
	ctx context.Context,
	curUser *model.User,
	query *bun.UpdateQuery,
) (*bun.UpdateQuery, error) {
	if curUser.Admin {
		return query, nil
	}
	userProjectsQ := db.Bun().NewSelect().Column("id").Table("projects").Where("user_id = ?", curUser.ID)
	query.Where("(user_id = ? OR project_id in (?))", curUser.ID, userProjectsQ)
	return query, nil
}

// DELETE /trial-comparison/collections

func (a *TrialAuthZBasic) AuthFilterCollectionsDeleteQuery(
	ctx context.Context,
	curUser *model.User,
	query *bun.DeleteQuery,
) (*bun.DeleteQuery, error) {
	if curUser.Admin {
		return query, nil
	}
	userProjectsQ := db.Bun().NewSelect().Column("id").Table("projects").Where("user_id = ?", curUser.ID)
	query.Where("(user_id = ? OR project_id in (?))", curUser.ID, userProjectsQ)
	return query, nil

}

// POST /trial-comparison/collections
func (a *TrialAuthZBasic) CanCreateTrialCollection(
	ctx context.Context, curUser *model.User, projectId int32,
) (canCreateTrialCollection bool, serverError error) {
	return true, nil
}

// POST /trial-comparison/query
// POST /trial-comparison/update-trial-tags
func (a *TrialAuthZBasic) AuthFilterTrialsQuery(
	ctx context.Context, curUser *model.User, query *bun.SelectQuery, update bool,
) (*bun.SelectQuery, error) {
	if update {
		// update case is analogous to CanEditExperimentsMetadata
		return query, nil
	}
	return query, nil

}

func init() {
	AuthZProvider.Register("basic", &TrialAuthZBasic{})
}

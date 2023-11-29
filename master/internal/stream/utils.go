package stream

import (
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// permFilter adds a filter to the provided bun query to filter for workspaces the user has
// access to.
func permFilter(
	q *bun.SelectQuery, accessMap model.AccessScopeSet, accessScopes []model.AccessScopeID,
) *bun.SelectQuery {
	if accessMap[model.GlobalAccessScopeID] {
		return q
	}
	return q.Where("workspace_id in (?)", bun.In(accessScopes))
}

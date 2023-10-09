package stream

import (
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

func permFilter(
	q *bun.SelectQuery, accessMap model.AccessScopeSet, accessScopes []model.AccessScopeID,
) *bun.SelectQuery {
	if accessMap[model.GlobalAccessScopeID] {
		return q
	}
	return q.Where("workspace_id in (?)", bun.In(accessScopes))
}

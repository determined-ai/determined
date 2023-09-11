package stream

import (
	"github.com/uptrace/bun"
)

type whereCall struct {
	query string
	args  []interface{}
}

// WhereSince makes it easier to write WHERE clauses of the following forms, which come up
// frequently in streaming:
// - WHERE seq > N AND <predicate>
// - WHERE seq > N AND (<predicate> OR <predicate> ...)
// - WHERE <predicate>
// - WHERE <predicate> OR <predicate>.
type WhereSince struct {
	// A filter on the seq column.  AND'ed with remaining predicates if nonzero.
	Since int64
	// Remaining calls are queued and process later, with ORs in between them.
	Includes []whereCall
}

// Include adds additional predicates to be appended together using ORs.
func (w *WhereSince) Include(query string, args ...interface{}) {
	w.Includes = append(w.Includes, whereCall{query, args})
}

func (w *WhereSince) chainIncludes(q *bun.SelectQuery) *bun.SelectQuery {
	where := q.Where
	for _, include := range w.Includes {
		q = where(include.query, include.args...)
		where = q.WhereOr
	}
	return q
}

// Apply applies the WhereSince caluse to the provided bun query.
func (w *WhereSince) Apply(q *bun.SelectQuery) *bun.SelectQuery {
	// Simple case first: either there's no sequence or only one predicate.
	// In those cases, no WhereGroup is needed.
	if w.Since < 1 || len(w.Includes) < 2 {
		if w.Since > 0 {
			q = q.Where("seq > ?", w.Since)
		}
		return w.chainIncludes(q)
	}

	// Complex case, we have a sequence and multiple predicates.
	q = q.Where("seq > ?", w.Since)
	return q.WhereGroup(" AND ", w.chainIncludes)
}

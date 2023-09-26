package bunutils

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// Paginate calculates pagination info for a Bun query and returns the SelectQuery for the
// caller to execute.
func Paginate(
	ctx context.Context, q *bun.SelectQuery, offset, limit int,
) (*bun.SelectQuery, *apiv1.Pagination, error) {
	// Count number of items without any limits or offsets.
	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Calculate end and start indexes.
	startIndex := offset
	if offset > total || offset < -total {
		startIndex = total
	} else if offset < 0 {
		startIndex = total + offset
	}

	endIndex := startIndex + limit
	switch {
	case limit == -2:
		endIndex = startIndex
	case limit == -1:
		endIndex = total
	case limit == 0:
		endIndex = 100 + startIndex
		if total < endIndex {
			endIndex = total
		}
	case startIndex+limit > total:
		endIndex = total
	}

	// Add start and end index to query.
	q.Offset(startIndex)
	q.Limit(endIndex - startIndex)

	return q, &apiv1.Pagination{
		Offset:     int32(offset),
		Limit:      int32(limit),
		Total:      int32(total),
		StartIndex: int32(startIndex),
		EndIndex:   int32(endIndex),
	}, nil
}

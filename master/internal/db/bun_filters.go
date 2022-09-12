package db

import (
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
)

// FilterComparison makes you wish for properties in generic structs/interfaces.
type FilterComparison[T any] struct {
	Gt  *T
	Gte *T
	Lt  *T
	Lte *T
}

func applyFieldFilterComparison[T any](
	q *bun.SelectQuery,
	column schema.Ident,
	filter FilterComparison[T],
) (*bun.SelectQuery, error) {
	if filter.Gt != nil {
		q = q.Where("? > ?", column, filter.Gt)
	}
	if filter.Gte != nil {
		q = q.Where("? >= ?", column, filter.Gte)
	}
	if filter.Lt != nil {
		q = q.Where("? < ?", column, filter.Lt)
	}
	if filter.Lte != nil {
		q = q.Where("? <= ?", column, filter.Lte)
	}
	return q, nil
}

// ApplyInt32FieldFilter applies filtering on a bun query for int32 field.
func ApplyInt32FieldFilter(
	q *bun.SelectQuery,
	column schema.Ident,
	filter *commonv1.Int32FieldFilter,
) (*bun.SelectQuery, error) {
	q, err := applyFieldFilterComparison(q, column, FilterComparison[int32]{
		Gt:  filter.Gt,
		Gte: filter.Gte,
		Lt:  filter.Lt,
		Lte: filter.Lte,
	})
	if err != nil {
		return nil, err
	}

	if filter.In != nil {
		values := []int32{}
		values = append(values, filter.In...)
		if len(values) == 0 {
			q = q.Where("false")
		} else {
			q = q.Where("? IN (?)", column, bun.In(values))
		}
	}

	if filter.NotIn != nil {
		values := []int32{}
		values = append(values, filter.NotIn...)
		if len(values) == 0 {
			q = q.Where("true")
		} else {
			q = q.Where("? NOT IN (?)", column, bun.In(values))
		}
	}

	return q, nil
}

// ApplyDoubleFieldFilter applies filtering on a bun query for double field.
func ApplyDoubleFieldFilter(
	q *bun.SelectQuery,
	column schema.Ident,
	filter *commonv1.DoubleFieldFilter,
) (*bun.SelectQuery, error) {
	q, err := applyFieldFilterComparison(q, column, FilterComparison[float64]{
		Gt:  filter.Gt,
		Gte: filter.Gte,
		Lt:  filter.Lt,
		Lte: filter.Lte,
	})
	if err != nil {
		return nil, err
	}

	return q, nil
}

func tryAsTime(tspb *timestamppb.Timestamp) *time.Time {
	if tspb == nil {
		return nil
	}

	return ptrs.Ptr(tspb.AsTime())
}

// ApplyTimestampFieldFilter applies filtering on a bun query for timestamp field.
func ApplyTimestampFieldFilter(
	q *bun.SelectQuery,
	column schema.Ident,
	filter *commonv1.TimestampFieldFilter,
) (*bun.SelectQuery, error) {
	q, err := applyFieldFilterComparison(q, column, FilterComparison[time.Time]{
		Gt:  tryAsTime(filter.Gt),
		Gte: tryAsTime(filter.Gte),
		Lt:  tryAsTime(filter.Lt),
		Lte: tryAsTime(filter.Lte),
	})
	if err != nil {
		return nil, err
	}

	return q, nil
}

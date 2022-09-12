package db

import (
	"strconv"
	"strings"
	"time"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FieldFilterComparison[T any] interface {
	GetGt() T
	GetGte() T
	GetLt() T
	GetLte() T
}

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

func parseCommaSeparatedInt32Value(value string) ([]int32, error) {
	arr := strings.Split(value, ",")
	result := []int32{}
	for i := range arr {
		parsed, err := strconv.Atoi(arr[i])
		if err != nil {
			return nil, err
		}
		result = append(result, int32(parsed))
	}
	return result, nil
}

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
		if *filter.In == "" {
			q = q.Where("false")
		} else {
			values, err := parseCommaSeparatedInt32Value(*filter.In)
			if err != nil {
				return nil, err
			}

			q = q.Where("? IN (?)", column, bun.In(values))
		}
	}

	if filter.NotIn != nil {
		if *filter.NotIn == "" {
			q = q.Where("true")
		} else {
			values, err := parseCommaSeparatedInt32Value(*filter.NotIn)
			if err != nil {
				return nil, err
			}

			q = q.Where("? NOT IN (?)", column, bun.In(values))
		}
	}

	return q, nil
}

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

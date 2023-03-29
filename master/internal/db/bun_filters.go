package db

import (
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/pkg/mathx"
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

func applyFieldFilterComparison[T any, U string | schema.Ident](
	q *bun.SelectQuery,
	column U,
	filter FilterComparison[T],
) (*bun.SelectQuery, error) {
	switch any(column).(type) {
	case string:
		if filter.Gt != nil {
			q = q.Where(fmt.Sprintf("%s > ?", column), filter.Gt)
		}
		if filter.Gte != nil {
			q = q.Where(fmt.Sprintf("%s >= ?", column), filter.Gte)
		}
		if filter.Lt != nil {
			q = q.Where(fmt.Sprintf("%s < ?", column), filter.Lt)
		}
		if filter.Lte != nil {
			q = q.Where(fmt.Sprintf("%s <= ?", column), filter.Lte)
		}
	default:
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
	}
	return q, nil
}

// ApplyInt32FieldFilter applies filtering on a bun query for int32 field.
func ApplyInt32FieldFilter[T string | schema.Ident](
	q *bun.SelectQuery,
	column T,
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

	if filter.Incl != nil {
		values := []int32{}
		values = append(values, filter.Incl...)
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
func ApplyDoubleFieldFilter[T string | schema.Ident](
	q *bun.SelectQuery,
	column T,
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
func ApplyTimestampFieldFilter[T string | schema.Ident](
	q *bun.SelectQuery,
	column T,
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

// TODO: Currently validations will still allow users to specify mutually exclusive ranges
// we may want to update logic to throw an error in such cases.

// ValidateInt32FieldFilterComparison validates the min and max values in the range.
func ValidateInt32FieldFilterComparison(
	filter *commonv1.Int32FieldFilter,
) error {
	var minValue, maxValue int32
	if filter == nil {
		return nil
	}
	if filter.Gt == nil && filter.Gte == nil {
		return nil
	}
	if filter.Lt == nil && filter.Lte == nil {
		return nil
	}
	if filter.Lt != nil && filter.Lte != nil { //nolint: gocritic
		maxValue = mathx.Max(*filter.Lt, *filter.Lte)
	} else if filter.Lt != nil {
		maxValue = *filter.Lt
	} else {
		maxValue = *filter.Lte
	}
	if filter.Gt != nil && filter.Gte != nil { //nolint: gocritic
		minValue = mathx.Min(*filter.Gt, *filter.Gte)
	} else if filter.Gt != nil {
		minValue = *filter.Gt
	} else {
		minValue = *filter.Gte
	}
	if minValue > maxValue {
		return fmt.Errorf("invalid integer range: min value %v cannot be larger than max value %v",
			minValue,
			maxValue,
		)
	}
	return nil
}

// ValidateTimeStampFieldFilterComparison validates the min and max timestamps in the range.
func ValidateTimeStampFieldFilterComparison(
	filter *commonv1.TimestampFieldFilter,
) error {
	var startTime, endTime time.Time
	if filter == nil {
		return nil
	}
	if filter.Gt == nil && filter.Gte == nil {
		return nil
	}
	if filter.Lt == nil && filter.Lte == nil {
		return nil
	}
	if filter.Lt != nil && filter.Lte != nil { //nolint: gocritic
		lt := tryAsTime(filter.Lt)
		lte := tryAsTime(filter.Lte)
		if lt.Before(*lte) {
			endTime = *lt
		} else {
			endTime = *lte
		}
	} else if filter.Lt != nil {
		endTime = *tryAsTime(filter.Lt)
	} else {
		endTime = *tryAsTime(filter.Lte)
	}
	if filter.Gt != nil && filter.Gte != nil { //nolint: gocritic
		gt := tryAsTime(filter.Gt)
		gte := tryAsTime(filter.Gte)
		if gt.After(*gte) {
			startTime = *gt
		} else {
			startTime = *gte
		}
	} else if filter.Lt != nil {
		startTime = *tryAsTime(filter.Lt)
	} else {
		startTime = *tryAsTime(filter.Lte)
	}
	if endTime.Before(startTime) {
		return fmt.Errorf("invalid date range: end date %v cannot be earlier than start date %v",
			endTime,
			startTime,
		)
	}
	return nil
}

// ValidateDoubleFieldFilterComparison validates the min and max values in the range.
func ValidateDoubleFieldFilterComparison(
	filter *commonv1.DoubleFieldFilter,
) error {
	var minValue, maxValue float64
	if filter == nil {
		return nil
	}
	if filter.Gt == nil && filter.Gte == nil {
		return nil
	}
	if filter.Lt == nil && filter.Lte == nil {
		return nil
	}
	if filter.Lt != nil && filter.Lte != nil { //nolint: gocritic
		maxValue = mathx.Max(*filter.Lt, *filter.Lte)
	} else if filter.Lt != nil {
		maxValue = *filter.Lt
	} else {
		maxValue = *filter.Lte
	}
	if filter.Gt != nil && filter.Gte != nil { //nolint: gocritic
		minValue = mathx.Min(*filter.Gt, *filter.Gte)
	} else if filter.Gt != nil {
		minValue = *filter.Gt
	} else {
		minValue = *filter.Gte
	}
	if minValue > maxValue {
		return fmt.Errorf("invalid double range: min value %v cannot be larger than max value %v",
			minValue,
			maxValue,
		)
	}
	return nil
}

// ValidatePolymorphicFilter ensures that a Polymorphic filter contains at most one valid range.
func ValidatePolymorphicFilter(
	filter *commonv1.PolymorphicFilter,
) error {
	if filter == nil {
		return nil
	}
	filterCount := 0
	if filter.TimeRange != nil {
		filterCount++
	}
	if filter.DoubleRange != nil {
		filterCount++
	}
	if filter.IntegerRange != nil {
		filterCount++
	}
	if filterCount > 1 {
		return fmt.Errorf(`invalid filter: only one filter 
		range may be specified however %v filters have beed defined`, filterCount)
	}
	if filter.TimeRange != nil {
		return ValidateTimeStampFieldFilterComparison(filter.TimeRange)
	}
	if filter.DoubleRange != nil {
		return ValidateDoubleFieldFilterComparison(filter.DoubleRange)
	}
	if filter.IntegerRange != nil {
		return ValidateInt32FieldFilterComparison(filter.IntegerRange)
	}
	return nil
}

// ApplyPolymorphicFilter applies filtering on a bun query for a polymorphic filter.
func ApplyPolymorphicFilter(
	q *bun.SelectQuery,
	column string,
	filter *commonv1.PolymorphicFilter,
) (*bun.SelectQuery, error) {
	if filter == nil {
		return q, nil
	}
	if filter.TimeRange != nil {
		return ApplyTimestampFieldFilter(q, column, filter.TimeRange)
	}
	if filter.IntegerRange != nil {
		return ApplyInt32FieldFilter(q, column, filter.IntegerRange)
	}
	if filter.DoubleRange != nil {
		return ApplyDoubleFieldFilter(q, column, filter.DoubleRange)
	}
	return q, nil
}

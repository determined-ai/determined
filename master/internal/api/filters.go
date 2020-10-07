package api

import (
	"fmt"

	"github.com/pkg/errors"

	filters "github.com/determined-ai/determined/proto/pkg/filtersv1"
)

// ErrUnsupportedFilter is the root cause of any filter validation failure.
var ErrUnsupportedFilter = errors.New("unsupported filter")

type defaultFilterError struct{}

func (e defaultFilterError) Cause() error {
	return ErrUnsupportedFilter
}

// ErrUnsupportedFilterField indicates the field in the filter is unsupported.
type ErrUnsupportedFilterField struct {
	Filter *filters.Filter
	defaultFilterError
}

func (e ErrUnsupportedFilterField) Error() string {
	return fmt.Sprintf("unsupported field in filter on %s", e.Filter.Field)
}

// ErrUnsupportedFilterOperation indicates the operation in the filter is unsupported.
type ErrUnsupportedFilterOperation struct {
	Filter *filters.Filter
	defaultFilterError
}

func (e ErrUnsupportedFilterOperation) Error() string {
	return fmt.Sprintf("unsupported operation %s for filter on %s", e.Filter.Operation, e.Filter.Field)
}

// ErrMissingFilterValues indicates the values are absent in the filter.
type ErrMissingFilterValues struct {
	Filter *filters.Filter
	defaultFilterError
}

func (e ErrMissingFilterValues) Error() string {
	return fmt.Sprintf("missing arguments for filter on %s", e.Filter.Field)
}

// ErrTooManyFilterValues indicates there are too many values for the field and operation requested.
type ErrTooManyFilterValues struct {
	Filter *filters.Filter
	defaultFilterError
}

func (e ErrTooManyFilterValues) Error() string {
	return fmt.Sprintf("wrong number of arguments for filter on %s", e.Filter.Field)
}

// ErrUnsupportedFilterValues indicates that the incorrect values were supplied.
type ErrUnsupportedFilterValues struct {
	Filter *filters.Filter
	defaultFilterError
}

func (e ErrUnsupportedFilterValues) Error() string {
	return fmt.Sprintf("unsupported values %T for filter on %s", e.Filter.Values, e.Filter.Field)
}

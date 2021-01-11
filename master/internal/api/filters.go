package api

// FilterOperation is an operation in a filter.
type FilterOperation int

const (
	// FilterOperationIn checks set membership.
	FilterOperationIn FilterOperation = iota
	// FilterOperationGreaterThan checks if the field is greater than a value.
	FilterOperationGreaterThan
	// FilterOperationLessThanEqual checks if the field is less than a value.
	FilterOperationLessThanEqual
)

// Filter is a general representation for a filter provided to an API.
type Filter struct {
	Field     string
	Operation FilterOperation
	Values    interface{}
}

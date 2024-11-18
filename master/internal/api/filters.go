package api

// FilterOperation is an operation in a filter.
type FilterOperation int

const (
	// FilterOperationIn checks set membership.
	FilterOperationIn FilterOperation = iota
	// FilterOperationInOrNull checks membership or a NULL option.
	FilterOperationInOrNull
	// FilterOperationGreaterThan checks if the field is greater than a value.
	FilterOperationGreaterThan
	// FilterOperationLessThanEqual checks if the field is less than or equal to a value.
	FilterOperationLessThanEqual
	// FilterOperationStringContainment checks if the field contains a value as a substring.
	FilterOperationStringContainment
)

// Filter is a general representation for a filter provided to an API.
type Filter struct {
	Field     string
	Operation FilterOperation
	Values    interface{}
}

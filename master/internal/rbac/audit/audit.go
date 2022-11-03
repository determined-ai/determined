package audit

import (
	"context"

	"github.com/sirupsen/logrus"
)

const (
	// GetMethod is the string for the get method.
	GetMethod = "get"
	// PostMethod is the string for the post method.
	PostMethod = "post"
	// DeleteMethod is the string for the delete method.
	DeleteMethod = "delete"
	// PutMethod is the string for the put method.
	PutMethod = "put"
)

// LogKey is the key used to store and extract logrus fields from context.
type LogKey struct{}

// ExtractLogFields retrieves logrus Fields from a context, if it exists.
func ExtractLogFields(ctx context.Context) logrus.Fields {
	fields := ctx.Value(LogKey{})
	if fields == nil {
		return logrus.Fields{}
	}
	logFields, ok := fields.(logrus.Fields)
	if !ok {
		return logrus.Fields{}
	}
	return logFields
}

package audit

import (
	"context"

	"github.com/sirupsen/logrus"
)

// LogKey is the key used to store and extract logrus fields from context.
type LogKey struct{}

// ExtractLogFields retrieves logrus Fields from a context, if it exists.
func ExtractLogFields(ctx context.Context) logrus.Fields {
	fields := ctx.Value(LogKey{})
	logFields, ok := fields.(logrus.Fields)
	if !ok {
		return logrus.Fields{}
	}
	return logFields
}

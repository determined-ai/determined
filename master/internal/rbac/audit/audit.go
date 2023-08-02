package audit

import (
	"context"

	"github.com/sirupsen/logrus"
)

// LogKey is the key used to store and extract logrus fields from context.
type LogKey struct{}

// EntityIDKey is the key used to store and extract entity IDs from log fields.
const EntityIDKey = "entityID"

// SupplyEntityID augments a context's log fields with the entity ID.
func SupplyEntityID(ctx context.Context, id interface{}) context.Context {
	logFields := ExtractLogFields(ctx)
	logFields[EntityIDKey] = id
	return context.WithValue(ctx, LogKey{}, logFields)
}

// ExtractLogFields retrieves logrus Fields from a context, if it exists.
func ExtractLogFields(ctx context.Context) logrus.Fields {
	fields := ctx.Value(LogKey{})
	logFields, ok := fields.(logrus.Fields)
	if !ok {
		return logrus.Fields{}
	}
	return logFields
}

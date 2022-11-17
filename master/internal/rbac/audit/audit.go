package audit

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/proto/pkg/rbacv1"
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

// PermissionWithSubject contains the permission and what subject requires it.
type PermissionWithSubject struct {
	PermissionTypes []rbacv1.PermissionType
	SubjectType     string
	SubjectIDs      []string
}

func (p PermissionWithSubject) String() string {
	switch {
	case len(p.PermissionTypes) == 0 && len(p.SubjectIDs) == 0:
		return fmt.Sprintf("operation on type %s", p.SubjectType)
	case len(p.PermissionTypes) == 0:
		return fmt.Sprintf("operation on type %s with IDs %s", p.SubjectType, p.SubjectIDs)
	case len(p.SubjectIDs) == 0:
		return fmt.Sprintf("operation on type %s requires the following permissions: %v",
			p.SubjectType, p.PermissionTypes)
	}

	return fmt.Sprintf("operation on type %s with IDs %s require the following permissions: %v",
		p.SubjectType, p.SubjectIDs, p.PermissionTypes)
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

// Log is a convenience function for logging to logrus.
func Log(fields logrus.Fields) {
	logrus.WithFields(fields).Info("RBAC Audit Logs")
}

// LogFromErr is a convenience function that interprets the error to determined whether
// permission was granted.
func LogFromErr(fields logrus.Fields, err error) {
	if err != nil {
		fields["permissionGranted"] = false
	} else {
		fields["permissionGranted"] = true
	}

	Log(fields)
}

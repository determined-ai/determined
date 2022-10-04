package db

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrNotEnoughPermissions is returned when a user
// does not have permissions required.
var ErrNotEnoughPermissions = status.Error(codes.PermissionDenied, "access denied")
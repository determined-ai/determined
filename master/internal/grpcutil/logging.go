package grpcutil

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
)

func grpcCodeToLogrusLevel(code codes.Code) logrus.Level {
	switch code {
	// TraceLevel: Provides very little new info, mostly noise.
	case codes.OK:
		return logrus.TraceLevel
	case codes.ResourceExhausted:
		return logrus.TraceLevel

	// DebugLevel: Mostly for developers.
	case codes.Canceled:
		return logrus.DebugLevel
	case codes.Unknown:
		return logrus.DebugLevel
	case codes.InvalidArgument:
		return logrus.DebugLevel
	case codes.NotFound:
		return logrus.DebugLevel
	case codes.AlreadyExists:
		return logrus.DebugLevel
	case codes.Unavailable:
		return logrus.DebugLevel
	case codes.Aborted:
		return logrus.DebugLevel
	case codes.OutOfRange:
		return logrus.DebugLevel

	// InfoLevel: Could be useful to sysadmins managing a cluster.
	case codes.DeadlineExceeded:
		return logrus.InfoLevel
	case codes.PermissionDenied:
		return logrus.InfoLevel
	case codes.Unauthenticated:
		return logrus.InfoLevel

	// ErrorLevel: Indicates a probable bug in the system.
	case codes.FailedPrecondition:
		return logrus.ErrorLevel
	case codes.Unimplemented:
		return logrus.ErrorLevel
	case codes.Internal:
		return logrus.ErrorLevel
	case codes.DataLoss:
		return logrus.ErrorLevel
	default:
		return logrus.ErrorLevel
	}
}

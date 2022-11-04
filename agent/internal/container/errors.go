package container

import (
	"github.com/determined-ai/determined/master/pkg/aproto"
)

// ErrMissing indicates a container was missing when we tried to reattach it after a crash.
var ErrMissing = &aproto.ContainerFailure{
	FailureType: aproto.ContainerMissing,
	ErrMsg:      "container is gone on reattachment",
}

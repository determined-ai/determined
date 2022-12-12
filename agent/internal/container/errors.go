package container

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/aproto"
)

// ErrMissing indicates a container was missing when we tried to reattach it after a crash.
var ErrMissing = aproto.NewContainerFailure(
	aproto.ContainerMissing,
	errors.New("container is gone on reattachment"),
)

// ErrKilledBeforeRun indicates a container was aborted before we were able to run it.
var ErrKilledBeforeRun = aproto.NewContainerFailure(
	aproto.ContainerAborted,
	errors.New("killed before run"),
)

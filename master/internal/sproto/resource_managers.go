package sproto

import (
	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type (
	// ConfigureEndpoints informs the resource manager to configure the endpoints resources.
	ConfigureEndpoints struct {
		System *actor.System
		Echo   *echo.Echo
	}

	// GetEndpointActorAddress requests the name of the actor that is managing the resources.
	GetEndpointActorAddress struct{}
)

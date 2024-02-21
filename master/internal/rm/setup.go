package rm

import (
	"crypto/tls"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/agentrm"
	"github.com/determined-ai/determined/master/internal/rm/kubernetesrm"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

// New sets up the actor and endpoints for resource managers.
func New(
	db *db.PgDB,
	echo *echo.Echo,
	config *config.ResourceConfig,
	taskContainerDefaults *model.TaskContainerDefaultsConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) ResourceManager {
	if len(config.ResourceManagers) != 1 {
		panic(fmt.Sprintf("expected only one resource manager got %d instead",
			len(config.ResourceManagers)))
	}

	// TODO(multirm) support multiple resource managers.
	c := config.ResourceManagers[0]
	switch {
	case c.AgentRM != nil:
		return agentrm.New(db, echo, config, opts, cert)
	case c.KubernetesRM != nil:
		return kubernetesrm.New(db, config, taskContainerDefaults, opts, cert)
	default:
		panic("no expected resource manager config is defined")
	}
}

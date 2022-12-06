package rm

import (
	"crypto/tls"

	"github.com/determined-ai/determined/master/internal/rm/agentrm"
	"github.com/determined-ai/determined/master/internal/rm/kubernetesrm"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
)

// New sets up the actor and endpoints for resource managers.
func New(
	system *actor.System,
	db *db.PgDB,
	echo *echo.Echo,
	config *config.ResourceConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) ResourceManager {
	switch {
	case config.ResourceManager.AgentRM != nil:
		return agentrm.New(system, db, echo, config, opts, cert)
	case config.ResourceManager.KubernetesRM != nil:
		return kubernetesrm.New(system, db, echo, config, opts, cert)
	default:
		panic("no expected resource manager config is defined")
	}
}

package internal

import (
	"crypto/tls"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/agentrm"
	"github.com/determined-ai/determined/master/internal/rm/kubernetesrm"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

// NewRM sets up the actor and endpoints for resource managers.
func NewRM(
	system *actor.System,
	db *db.PgDB,
	echo *echo.Echo,
	config *config.ResourceConfig,
	taskContainerDefaults *model.TaskContainerDefaultsConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) rm.ResourceManager {
	switch {
	case config.ResourceManager.AgentRM != nil:
		return agentrm.New(system, db, echo, config, opts, cert)
	case config.ResourceManager.KubernetesRM != nil:
		return kubernetesrm.New(system, db, echo, config, taskContainerDefaults, opts, cert)
	default:
		panic("no expected resource manager config is defined")
	}
}

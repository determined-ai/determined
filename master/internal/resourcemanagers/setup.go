package resourcemanagers

import (
	"crypto/tls"

	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/agent"
	"github.com/determined-ai/determined/master/internal/kubernetes"
	"github.com/determined-ai/determined/master/pkg/actor"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/model"
)

// Setup setups the actor and endpoints for resource managers.
func Setup(
	system *actor.System,
	echo *echo.Echo,
	rmConfig *ResourceManagerConfig,
	poolsConfig *ResourcePoolsConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) *actor.Ref {
	var ref *actor.Ref
	switch {
	case rmConfig.AgentRM != nil:
		ref = setupAgentResourceManager(system, echo, rmConfig.AgentRM, poolsConfig, opts, cert)
	case rmConfig.KubernetesRM != nil:
		ref = setupKubernetesResourceManager(system, echo, rmConfig.KubernetesRM, opts.LoggingOptions)
	default:
		panic("no expected resource manager config is defined")
	}

	rm, ok := system.ActorOf(actor.Addr("resourceManagers"), &ResourceManagers{ref: ref})
	if !ok {
		panic("cannot create resource managers")
	}
	return rm
}

func setupAgentResourceManager(
	system *actor.System,
	echo *echo.Echo,
	rmConfig *AgentResourceManagerConfig,
	poolsConfig *ResourcePoolsConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) *actor.Ref {
	ref, _ := system.ActorOf(
		actor.Addr("agentRM"),
		newAgentResourceManager(rmConfig, poolsConfig, cert),
	)
	system.Ask(ref, actor.Ping{}).Get()

	logrus.Infof("initializing endpoints for agents")
	agent.Initialize(system, echo, opts)
	return ref
}

func setupKubernetesResourceManager(
	system *actor.System,
	echo *echo.Echo,
	config *KubernetesResourceManagerConfig,
	loggingConfig model.LoggingConfig,
) *actor.Ref {
	ref, _ := system.ActorOf(
		actor.Addr("kubernetesRM"),
		newKubernetesResourceManager(config),
	)
	system.Ask(ref, actor.Ping{}).Get()

	logrus.Infof("initializing endpoints for pods")
	kubernetes.Initialize(
		system, echo, ref, config.Namespace, config.MasterServiceName, loggingConfig,
		config.LeaveKubernetesResources,
	)
	return ref
}

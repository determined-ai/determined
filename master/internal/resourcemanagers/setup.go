package resourcemanagers

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"

	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/agent"
	"github.com/determined-ai/determined/master/internal/kubernetes"
	"github.com/determined-ai/determined/master/pkg/actor"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/model"
)

func makeTLSConfig(cert *tls.Certificate) model.TLSClientConfig {
	if cert == nil {
		return model.TLSClientConfig{}
	}
	var content bytes.Buffer
	for _, c := range cert.Certificate {
		// Errors can only happen due to invalid headers (of which there are none) or I/O (which is safe
		// with a bytes.Buffer).
		_ = pem.Encode(&content, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: c,
		})
	}

	leaf, _ := x509.ParseCertificate(cert.Certificate[0])
	certName := ""
	if len(leaf.DNSNames) > 0 {
		certName = leaf.DNSNames[0]
	} else if len(leaf.IPAddresses) > 0 {
		certName = leaf.IPAddresses[0].String()
	}

	return model.TLSClientConfig{
		Enabled:         true,
		CertBytes:       content.Bytes(),
		CertificateName: certName,
	}
}

// Setup sets up the actor and endpoints for resource managers.
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
		ref = setupKubernetesResourceManager(
			system, echo, rmConfig.KubernetesRM, makeTLSConfig(cert), opts.LoggingOptions,
		)
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
	masterTLSConfig model.TLSClientConfig,
	loggingConfig model.LoggingConfig,
) *actor.Ref {
	ref, _ := system.ActorOf(
		actor.Addr("kubernetesRM"),
		newKubernetesResourceManager(config),
	)
	system.Ask(ref, actor.Ping{}).Get()

	logrus.Infof("initializing endpoints for pods")
	kubernetes.Initialize(
		system, echo, ref, config.Namespace, config.MasterServiceName, masterTLSConfig, loggingConfig,
		config.LeaveKubernetesResources,
	)
	return ref
}

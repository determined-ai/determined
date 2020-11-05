package container

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"

	"github.com/determined-ai/determined/master/pkg/archive"
)

// Spec provides the necessary information for an agent to start a container.
type Spec struct {
	PullSpec PullSpec
	RunSpec  RunSpec
}

// PullSpec contains configs for an ImagePull call.
type PullSpec struct {
	ForcePull bool
	Registry  *types.AuthConfig
}

// RunSpec contains configs for ContainerCreate, CopyToContainer, and ContainerStart calls.
type RunSpec struct {
	ContainerConfig  container.Config
	HostConfig       container.HostConfig
	NetworkingConfig network.NetworkingConfig
	ChecksConfig     ChecksConfig

	Archives         []RunArchive
	UseFluentLogging bool
}

// ChecksConfig describes the configuration for multiple readiness checks.
type ChecksConfig struct {
	// PeriodSeconds is how long in seconds to wait between successive checks.
	PeriodSeconds float64
	// Checks describes all the checks that must pass for a container to be considered ready.
	Checks []CheckConfig
}

// CheckConfig describes the configuration for an HTTP readiness check.
type CheckConfig struct {
	// Port specifies the port inside the container that the service is listening on.
	Port int
	// Path specifies the path to request over HTTP (not including the '/' right after the host/port).
	Path string
}

// RunArchive contains one set of files sent over per CopyToContainer call.
type RunArchive struct {
	Path        string
	Archive     archive.Archive
	CopyOptions types.CopyToContainerOptions
}

package dockerflags

import (
	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

// Parse runs same parsing as "docker run" to get Docker SDK structs.
func Parse(
	args []string,
) (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
	if len(args) == 0 {
		return &container.Config{}, &container.HostConfig{}, &network.NetworkingConfig{}, nil
	}

	flagSet := pflag.NewFlagSet("parse", pflag.ContinueOnError)
	cOptions := addFlags(flagSet)
	if err := flagSet.Parse(args); err != nil {
		return nil, nil, nil, err
	}

	res, err := parse(flagSet, cOptions, "linux")
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error parsing docker flags")
	}

	processConfig(res)
	return res.Config, res.HostConfig, res.NetworkingConfig, nil
}

func processConfig(res *containerConfig) {
	// Attaching stdout and stderr doesn't make sense for our system.
	res.Config.AttachStdin = false
	res.Config.AttachStdout = false
	res.Config.AttachStderr = false

	// Zero out values to make them exactly same as default initialization.
	if len(res.Config.ExposedPorts) == 0 {
		res.Config.ExposedPorts = nil
	}
	if len(res.Config.Volumes) == 0 {
		res.Config.Volumes = nil
	}
	if len(res.Config.Labels) == 0 {
		res.Config.Labels = nil
	}

	if len(res.HostConfig.LogConfig.Config) == 0 {
		res.HostConfig.LogConfig.Config = nil
	}
	if len(res.HostConfig.PortBindings) == 0 {
		res.HostConfig.PortBindings = nil
	}
	if len(res.HostConfig.DNS) == 0 {
		res.HostConfig.DNS = nil
	}
	if len(res.HostConfig.DNSOptions) == 0 {
		res.HostConfig.DNSOptions = nil
	}
	if len(res.HostConfig.DNSSearch) == 0 {
		res.HostConfig.DNSSearch = nil
	}
	if len(res.HostConfig.StorageOpt) == 0 {
		res.HostConfig.StorageOpt = nil
	}
	if len(res.HostConfig.Tmpfs) == 0 {
		res.HostConfig.Tmpfs = nil
	}
	if len(res.HostConfig.Sysctls) == 0 {
		res.HostConfig.Sysctls = nil
	}
	if len(res.HostConfig.BlkioWeightDevice) == 0 {
		res.HostConfig.BlkioWeightDevice = nil
	}
	if len(res.HostConfig.Devices) == 0 {
		res.HostConfig.Devices = nil
	}
	if res.HostConfig.MemorySwappiness != nil && *res.HostConfig.MemorySwappiness == -1 {
		res.HostConfig.MemorySwappiness = nil
	}
	if res.HostConfig.MemorySwappiness != nil && *res.HostConfig.MemorySwappiness == -1 {
		res.HostConfig.MemorySwappiness = nil
	}
	if res.HostConfig.PidsLimit != nil && *res.HostConfig.PidsLimit == 0 {
		res.HostConfig.PidsLimit = nil
	}
	if res.HostConfig.OomKillDisable != nil && !*res.HostConfig.OomKillDisable {
		res.HostConfig.OomKillDisable = nil
	}
	if res.HostConfig.RestartPolicy.Name == "no" &&
		res.HostConfig.RestartPolicy.MaximumRetryCount == 0 {
		res.HostConfig.RestartPolicy = container.RestartPolicy{}
	}
	if res.HostConfig.NetworkMode == "default" {
		res.HostConfig.NetworkMode = ""
	}

	if len(res.NetworkingConfig.EndpointsConfig) == 0 {
		res.NetworkingConfig.EndpointsConfig = nil
	}
}

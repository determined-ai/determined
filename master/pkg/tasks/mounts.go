package tasks

import (
	"path/filepath"

	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// ToDockerMounts converts expconf bind mounts to container mounts.
func ToDockerMounts(bindMounts []expconf.BindMount) []mount.Mount {
	dockerMounts := make([]mount.Mount, 0, len(bindMounts))
	for _, m := range bindMounts {
		target := m.ContainerPath
		if !filepath.IsAbs(target) {
			target = filepath.Join(ContainerWorkDir, target)
		}
		dockerMounts = append(dockerMounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   m.HostPath,
			Target:   target,
			ReadOnly: *m.ReadOnly,
			BindOptions: &mount.BindOptions{
				Propagation: mount.Propagation(*m.Propagation),
			},
		})
	}
	return dockerMounts
}

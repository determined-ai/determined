package tasks

import (
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
)

// CommandSpec is a description of a task for running a command.
type CommandSpec struct {
	Config          model.CommandConfig
	UserFiles       archive.Archive
	AdditionalFiles archive.Archive
}

// ToTaskSpec generates a TaskSpec.
func (s CommandSpec) ToTaskSpec(base TaskSpec) TaskSpec {
	res := base

	res.Archives = base.makeArchives([]container.RunArchive{
		wrapArchive(base.AgentUserGroup.OwnArchive(s.UserFiles), ContainerWorkDir),
		wrapArchive(s.AdditionalFiles, rootDir),
	})

	res.Description = "cmd"

	res.Entrypoint = s.Config.Entrypoint

	res.Environment = s.Config.Environment.ToExpconf()

	res.EnvVars = base.makeEnvVars(nil)

	res.Mounts = ToDockerMounts(s.Config.BindMounts.ToExpconf())

	if shm := s.Config.Resources.ShmSize; shm != nil {
		res.ShmSize = int64(*shm)
	}
	res.ShmSize = 0

	res.ResourcesConfig = s.Config.Resources.ToExpconf()

	return res
}

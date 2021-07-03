package tasks

import (
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
)

// CommandSpec is a description of a task for running a command.
type CommandSpec struct {
	Base TaskSpec

	Config          model.CommandConfig
	UserFiles       archive.Archive
	AdditionalFiles archive.Archive
	Metadata        map[string]interface{}
}

// ToTaskSpec generates a TaskSpec.
func (s CommandSpec) ToTaskSpec() TaskSpec {
	res := s.Base

	res.Archives = s.Base.makeArchives([]container.RunArchive{
		wrapArchive(s.Base.AgentUserGroup.OwnArchive(s.UserFiles), ContainerWorkDir),
		wrapArchive(s.AdditionalFiles, rootDir),
	})

	res.Description = "cmd"

	res.Entrypoint = s.Config.Entrypoint

	res.Environment = s.Config.Environment.ToExpconf()

	res.EnvVars = s.Base.makeEnvVars(nil)

	res.Mounts = ToDockerMounts(s.Config.BindMounts.ToExpconf())

	if shm := s.Config.Resources.ShmSize; shm != nil {
		res.ShmSize = int64(*shm)
	}
	res.ShmSize = 0

	res.ResourcesConfig = s.Config.Resources.ToExpconf()

	return res
}

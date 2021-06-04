package tasks

import (
	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// StartCommand is a description of a task for running a command.
type StartCommand struct {
	Config          model.CommandConfig
	UserFiles       archive.Archive
	AdditionalFiles archive.Archive
}

// Archives implements InnerSpec.
func (s StartCommand) Archives(u *model.AgentUserGroup) []container.RunArchive {
	return []container.RunArchive{
		wrapArchive(u.OwnArchive(s.UserFiles), ContainerWorkDir),
		wrapArchive(s.AdditionalFiles, rootDir),
	}
}

// Description implements InnerSpec.
func (s StartCommand) Description() string { return "cmd" }

// Entrypoint implements InnerSpec.
func (s StartCommand) Entrypoint() []string { return s.Config.Entrypoint }

// Environment implements InnerSpec.
func (s StartCommand) Environment(TaskSpec) expconf.EnvironmentConfig {
	return s.Config.Environment.ToExpconf()
}

// EnvVars implements InnerSpec.
func (s StartCommand) EnvVars(TaskSpec) map[string]string { return nil }

// LoggingFields implements InnerSpec.
func (s StartCommand) LoggingFields() map[string]string { return nil }

// Mounts implements InnerSpec.
func (s StartCommand) Mounts() []mount.Mount {
	return ToDockerMounts(s.Config.BindMounts.ToExpconf())
}

// ShmSize implements InnerSpec.
func (s StartCommand) ShmSize() int64 {
	if shm := s.Config.Resources.ShmSize; shm != nil {
		return int64(*shm)
	}
	return 0
}

// UseFluentLogging implements InnerSpec.
func (s StartCommand) UseFluentLogging() bool { return false }

// UseHostMode implements InnerSpec.
func (s StartCommand) UseHostMode() bool { return false }

// ResourcesConfig implements InnerSpec.
func (s StartCommand) ResourcesConfig() expconf.ResourcesConfig {
	return s.Config.Resources.ToExpconf()
}

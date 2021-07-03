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

// ExtraArchives implements TaskContainer.
func (s StartCommand) ExtraArchives(u *model.AgentUserGroup) []container.RunArchive {
	return []container.RunArchive{
		wrapArchive(u.OwnArchive(s.UserFiles), ContainerWorkDir),
		wrapArchive(s.AdditionalFiles, rootDir),
	}
}

// Description implements TaskContainer.
func (s StartCommand) Description() string { return "cmd" }

// Entrypoint implements TaskContainer.
func (s StartCommand) Entrypoint() []string { return s.Config.Entrypoint }

// Environment implements TaskContainer.
func (s StartCommand) Environment() expconf.EnvironmentConfig {
	return s.Config.Environment.ToExpconf()
}

// ExtraEnvVars implements TaskContainer.
func (s StartCommand) ExtraEnvVars() map[string]string { return nil }

// LoggingFields implements TaskContainer.
func (s StartCommand) LoggingFields() map[string]string { return nil }

// Mounts implements TaskContainer.
func (s StartCommand) Mounts() []mount.Mount {
	return ToDockerMounts(s.Config.BindMounts.ToExpconf())
}

// ShmSize implements TaskContainer.
func (s StartCommand) ShmSize() int64 {
	if shm := s.Config.Resources.ShmSize; shm != nil {
		return int64(*shm)
	}
	return 0
}

// UseFluentLogging implements TaskContainer.
func (s StartCommand) UseFluentLogging() bool { return false }

// UseHostMode implements TaskContainer.
func (s StartCommand) UseHostMode() bool { return false }

// ResourcesConfig implements TaskContainer.
func (s StartCommand) ResourcesConfig() expconf.ResourcesConfig {
	return s.Config.Resources.ToExpconf()
}

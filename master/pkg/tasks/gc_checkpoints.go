package tasks

import (
	"archive/tar"
	"encoding/json"
	"path/filepath"
	"strconv"

	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// GCCheckpoints is a description of a task for running checkpoint GC.
type GCCheckpoints struct {
	ExperimentID       int
	LegacyConfig       expconf.LegacyConfig
	ToDelete           json.RawMessage
	DeleteTensorboards bool
}

// Archives implements InnerSpec.
func (g GCCheckpoints) Archives(u *model.AgentUserGroup) []container.RunArchive {
	return []container.RunArchive{
		wrapArchive(
			archive.Archive{
				u.OwnedArchiveItem(
					"storage_config.json",
					[]byte(jsonify(g.LegacyConfig.CheckpointStorage())),
					0600,
					tar.TypeReg,
				),
				u.OwnedArchiveItem(
					"checkpoints_to_delete.json",
					[]byte(jsonify(g.ToDelete)),
					0600,
					tar.TypeReg,
				),
				u.OwnedArchiveItem(
					etc.GCCheckpointsEntrypointResource,
					etc.MustStaticFile(etc.GCCheckpointsEntrypointResource),
					0700,
					tar.TypeReg,
				),
			},
			ContainerWorkDir,
		),
	}
}

// Description implements InnerSpec.
func (g GCCheckpoints) Description() string { return "gc" }

// Entrypoint implements InnerSpec.
func (g GCCheckpoints) Entrypoint() []string {
	e := []string{
		filepath.Join(ContainerWorkDir, etc.GCCheckpointsEntrypointResource),
		"--experiment-id",
		strconv.Itoa(g.ExperimentID),
		"--storage-config",
		"storage_config.json",
		"--delete",
		"checkpoints_to_delete.json",
	}
	if g.DeleteTensorboards {
		e = append(e, "--delete-tensorboards")
	}
	return e
}

// Environment implements InnerSpec.
func (g GCCheckpoints) Environment(t TaskSpec) expconf.EnvironmentConfig {
	// Keep only the EnvironmentVariables provided by the experiment's config.
	envvars := g.LegacyConfig.EnvironmentVariables()
	env := expconf.EnvironmentConfig{
		RawEnvironmentVariables: &envvars,
	}

	// Fill the rest of the environment with default values.
	defaultConfig := expconf.ExperimentConfig{}
	t.TaskContainerDefaults.MergeIntoConfig(&defaultConfig)

	if defaultConfig.RawEnvironment != nil {
		env = schemas.Merge(env, *defaultConfig.RawEnvironment).(expconf.EnvironmentConfig)
	}
	return schemas.WithDefaults(env).(expconf.EnvironmentConfig)
}

// EnvVars implements InnerSpec.
func (g GCCheckpoints) EnvVars(TaskSpec) map[string]string { return nil }

// LoggingFields implements InnerSpec.
func (g GCCheckpoints) LoggingFields() map[string]string { return nil }

// Mounts implements InnerSpec.
func (g GCCheckpoints) Mounts() []mount.Mount {
	mounts := ToDockerMounts(g.LegacyConfig.BindMounts())
	if fs := g.LegacyConfig.CheckpointStorage().RawSharedFSConfig; fs != nil {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: fs.HostPath(),
			Target: model.DefaultSharedFSContainerPath,
			BindOptions: &mount.BindOptions{
				Propagation: model.DefaultSharedFSPropagation,
			},
		})
	}
	return mounts
}

// ShmSize implements InnerSpec.
func (g GCCheckpoints) ShmSize() int64 { return 0 }

// UseFluentLogging implements InnerSpec.
func (g GCCheckpoints) UseFluentLogging() bool { return false }

// UseHostMode implements InnerSpec.
func (g GCCheckpoints) UseHostMode() bool { return false }

// ResourcesConfig implements InnerSpec.
func (g GCCheckpoints) ResourcesConfig() expconf.ResourcesConfig {
	// The GCCheckpoints resources config is effictively unused, so we return an empty one.
	return expconf.ResourcesConfig{}
}

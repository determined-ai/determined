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

// GCCkptSpec is a description of a task for running checkpoint GC.
type GCCkptSpec struct {
	Base TaskSpec

	ExperimentID       int
	LegacyConfig       expconf.LegacyConfig
	ToDelete           json.RawMessage
	DeleteTensorboards bool
}

// ToTaskSpec generates a TaskSpec.
func (g GCCkptSpec) ToTaskSpec(allocationToken string) TaskSpec {
	res := g.Base

	res.AllocationSessionToken = allocationToken

	res.ExtraArchives = []container.RunArchive{
		wrapArchive(
			archive.Archive{
				g.Base.AgentUserGroup.OwnedArchiveItem(
					"storage_config.json",
					[]byte(jsonify(g.LegacyConfig.CheckpointStorage())),
					0600,
					tar.TypeReg,
				),
				g.Base.AgentUserGroup.OwnedArchiveItem(
					"checkpoints_to_delete.json",
					[]byte(jsonify(g.ToDelete)),
					0600,
					tar.TypeReg,
				),
				g.Base.AgentUserGroup.OwnedArchiveItem(
					etc.GCCheckpointsEntrypointResource,
					etc.MustStaticFile(etc.GCCheckpointsEntrypointResource),
					0700,
					tar.TypeReg,
				),
			},
			ContainerWorkDir,
		),
	}

	res.Description = "gc"

	res.Entrypoint = []string{
		filepath.Join(ContainerWorkDir, etc.GCCheckpointsEntrypointResource),
		"--experiment-id",
		strconv.Itoa(g.ExperimentID),
		"--storage-config",
		"storage_config.json",
		"--delete",
		"checkpoints_to_delete.json",
	}
	if g.DeleteTensorboards {
		res.Entrypoint = append(res.Entrypoint, "--delete-tensorboards")
	}

	// Keep only the EnvironmentVariables provided by the experiment's config.
	envVars := g.LegacyConfig.EnvironmentVariables()
	env := expconf.EnvironmentConfig{
		RawEnvironmentVariables: &envVars,
	}
	// Fill the rest of the environment with default values.
	defaultConfig := expconf.ExperimentConfig{}
	g.Base.TaskContainerDefaults.MergeIntoExpConfig(&defaultConfig)

	if defaultConfig.RawEnvironment != nil {
		env = schemas.Merge(env, *defaultConfig.RawEnvironment).(expconf.EnvironmentConfig)
	}
	res.Environment = schemas.WithDefaults(env).(expconf.EnvironmentConfig)

	res.Mounts = ToDockerMounts(g.LegacyConfig.BindMounts())
	if fs := g.LegacyConfig.CheckpointStorage().RawSharedFSConfig; fs != nil {
		res.Mounts = append(res.Mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: fs.HostPath(),
			Target: model.DefaultSharedFSContainerPath,
			BindOptions: &mount.BindOptions{
				Propagation: model.DefaultSharedFSPropagation,
			},
		})
	}

	return res
}

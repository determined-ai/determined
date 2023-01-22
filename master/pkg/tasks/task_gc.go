package tasks

import (
	"archive/tar"
	"path/filepath"
	"strconv"

	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
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
	ToDelete           string
	DeleteTensorboards bool
}

// ToTaskSpec generates a TaskSpec.
func (g GCCkptSpec) ToTaskSpec() TaskSpec {
	res := g.Base

	// Set Environment.
	// Keep only the EnvironmentVariables provided by the experiment's config.
	envVars := g.LegacyConfig.Environment.EnvironmentVariables()
	//nolint:exhaustivestruct // This has caused an issue before, but is valid as a partial struct.
	env := expconf.EnvironmentConfig{
		RawEnvironmentVariables: &envVars,
		RawPodSpec:              g.LegacyConfig.Environment.PodSpec(),
	}
	// Fill the rest of the environment with default values.
	var defaultConfig expconf.ExperimentConfig
	g.Base.TaskContainerDefaults.MergeIntoExpConfig(&defaultConfig)
	if defaultConfig.RawEnvironment != nil {
		env = schemas.Merge(env, *defaultConfig.RawEnvironment)
	}
	res.Environment = schemas.WithDefaults(env)
	res.ExtraEnvVars = map[string]string{"DET_TASK_TYPE": string(model.TaskTypeCheckpointGC)}

	res.WorkDir = DefaultWorkDir

	res.ExtraArchives = []cproto.RunArchive{
		wrapArchive(
			archive.Archive{
				g.Base.AgentUserGroup.OwnedArchiveItem("checkpoint_gc", nil, 0o700, tar.TypeDir),
				g.Base.AgentUserGroup.OwnedArchiveItem(
					"checkpoint_gc/storage_config.json",
					[]byte(jsonify(g.LegacyConfig.CheckpointStorage)),
					0o600,
					tar.TypeReg,
				),
				g.Base.AgentUserGroup.OwnedArchiveItem(
					"checkpoint_gc/checkpoints_to_delete.json",
					[]byte(jsonify(g.ToDelete)),
					0o600,
					tar.TypeReg,
				),
				g.Base.AgentUserGroup.OwnedArchiveItem(
					filepath.Join("checkpoint_gc", etc.GCCheckpointsEntrypointResource),
					etc.MustStaticFile(etc.GCCheckpointsEntrypointResource),
					0o700,
					tar.TypeReg,
				),
			},
			runDir,
		),
	}

	res.Description = "gc"

	res.Entrypoint = []string{
		filepath.Join("/run/determined/checkpoint_gc", etc.GCCheckpointsEntrypointResource),
		"--experiment-id",
		strconv.Itoa(g.ExperimentID),
		"--storage-config",
		"/run/determined/checkpoint_gc/storage_config.json",
	}
	if g.ToDelete != "" {
		res.Entrypoint = append(res.Entrypoint, "--delete")
		res.Entrypoint = append(res.Entrypoint, g.ToDelete)
	}
	if g.DeleteTensorboards {
		res.Entrypoint = append(res.Entrypoint, "--delete-tensorboards")
	}

	res.Mounts = ToDockerMounts(g.LegacyConfig.BindMounts, res.WorkDir)
	if fs := g.LegacyConfig.CheckpointStorage.RawSharedFSConfig; fs != nil {
		res.Mounts = append(res.Mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: fs.HostPath(),
			Target: expconf.DefaultSharedFSContainerPath,
			BindOptions: &mount.BindOptions{
				Propagation: expconf.DefaultSharedFSPropagation,
			},
		})
	}
	res.TaskType = model.TaskTypeCheckpointGC

	return res
}

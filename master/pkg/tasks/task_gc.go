package tasks

import (
	"archive/tar"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

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

	ExperimentID int
	LegacyConfig expconf.LegacyConfig
	ToDelete     string
	// If len(CheckpointGlobs) == 0 then we won't delete any checkpoint files
	// and just refresh the state of the checkpoint.
	CheckpointGlobs   []string
	DeletedExperiment bool
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
	res.ResourcesConfig = schemas.WithDefaults(res.ResourcesConfig)
	res.SlurmConfig = defaultConfig.SlurmConfig()
	res.PbsConfig = defaultConfig.PbsConfig()

	res.WorkDir = DefaultWorkDir

	globs := g.CheckpointGlobs
	if globs == nil { // This matters for JSON parsing as [] vs None.
		globs = []string{}
	}

	storageConfigPath := "checkpoint_gc/storage_config.json"
	checkpointsToDeletePath := "checkpoint_gc/checkpoints_to_delete.json"
	checkpointsGlobsPath := "checkpoint_gc/checkpoints_globs.json"
	res.ExtraArchives = []cproto.RunArchive{
		wrapArchive(
			archive.Archive{
				g.Base.AgentUserGroup.OwnedArchiveItem("checkpoint_gc", nil, 0o700, tar.TypeDir),
				g.Base.AgentUserGroup.OwnedArchiveItem(
					storageConfigPath,
					[]byte(jsonify(g.LegacyConfig.CheckpointStorage)),
					0o600,
					tar.TypeReg,
				),
				g.Base.AgentUserGroup.OwnedArchiveItem(
					checkpointsToDeletePath,
					[]byte(jsonify(strings.Split(g.ToDelete, ","))),
					0o600,
					tar.TypeReg,
				),
				g.Base.AgentUserGroup.OwnedArchiveItem(
					checkpointsGlobsPath,
					[]byte(jsonify(globs)),
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
			RunDir,
		),
	}

	res.Description = fmt.Sprintf("gc-%d", g.ExperimentID)

	// We pass storage-config / delete / globs through a JSON file instead of a JSON string
	// to avoid reaching any OS limitations on sizes of CLI arguments.
	res.Entrypoint = []string{
		filepath.Join("/run/determined/checkpoint_gc", etc.GCCheckpointsEntrypointResource),
		"--experiment-id",
		strconv.Itoa(g.ExperimentID),
		"--storage-config", fmt.Sprintf("/run/determined/%s", storageConfigPath),
		"--delete", fmt.Sprintf("/run/determined/%s", checkpointsToDeletePath),
		"--globs", fmt.Sprintf("/run/determined/%s", checkpointsGlobsPath),
	}
	if g.DeletedExperiment {
		res.Entrypoint = append(res.Entrypoint, "--deleted-experiment")
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

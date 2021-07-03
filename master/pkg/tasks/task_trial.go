package tasks

import (
	"archive/tar"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/workload"
)

// TrialSpec is a description of a task for running a trial container.
type TrialSpec struct {
	ExperimentConfig    expconf.ExperimentConfig
	ModelDefinition     archive.Archive
	HParams             map[string]interface{}
	TrialSeed           uint32
	LatestCheckpoint    *model.Checkpoint
	InitialWorkload     workload.Workload
	WorkloadManagerType model.WorkloadManagerType
	AdditionalFiles     archive.Archive

	// This is used to hint the resource manager to override defaults and start
	// the container in host mode iff it has been scheduled across multiple agents.
	IsMultiAgent bool

	Rank int
}

// ToTaskSpec generates a TaskSpec.
func (s TrialSpec) ToTaskSpec(base TaskSpec) TaskSpec {
	res := base

	res.Archives = base.makeArchives([]container.RunArchive{
		wrapArchive(
			archive.Archive{
				base.AgentUserGroup.OwnedArchiveItem(trainDir, nil, 0700, tar.TypeDir),
				base.AgentUserGroup.OwnedArchiveItem(modelCopy, nil, 0700, tar.TypeDir),
			},
			rootDir,
		),
		wrapArchive(s.AdditionalFiles, rootDir),
		wrapArchive(
			archive.Archive{
				base.AgentUserGroup.OwnedArchiveItem(
					"checkpoint.json",
					[]byte(jsonify(s.LatestCheckpoint)),
					0600,
					tar.TypeReg,
				),
			},
			trainDir,
		),
		wrapArchive(base.AgentUserGroup.OwnArchive(s.ModelDefinition), modelCopy),
		wrapArchive(base.AgentUserGroup.OwnArchive(s.ModelDefinition), ContainerWorkDir),
	})

	res.Description = fmt.Sprintf(
		"exp-%d-trial-%d-rank-%d",
		s.InitialWorkload.ExperimentID,
		s.InitialWorkload.TrialID,
		s.Rank,
	)

	res.Entrypoint = []string{"/run/determined/train/entrypoint.sh"}

	env := s.ExperimentConfig.Environment()
	ports := env.Ports()
	if ports == nil {
		ports = make(map[string]int)
	}
	ports["trial"] = rendezvousPort(trialUniquePortOffset(base.Devices))
	env.SetPorts(ports)
	res.Environment = env

	portOffset := trialUniquePortOffset(base.Devices)
	portStr := rendezvousPort(portOffset)
	envVars := map[string]string{
		"DET_EXPERIMENT_ID":            fmt.Sprintf("%d", s.InitialWorkload.ExperimentID),
		"DET_TRIAL_ID":                 fmt.Sprintf("%d", s.InitialWorkload.TrialID),
		"DET_TRIAL_SEED":               fmt.Sprintf("%d", s.TrialSeed),
		"DET_EXPERIMENT_CONFIG":        jsonify(s.ExperimentConfig),
		"DET_HPARAMS":                  jsonify(s.HParams),
		"DET_INITIAL_WORKLOAD":         jsonify(s.InitialWorkload),
		"DET_LATEST_CHECKPOINT":        "/run/determined/train/checkpoint.json",
		"DET_WORKLOAD_MANAGER_TYPE":    string(s.WorkloadManagerType),
		"DET_RENDEZVOUS_PORT":          strconv.Itoa(portStr),
		"DET_TRIAL_UNIQUE_PORT_OFFSET": strconv.Itoa(portOffset),
	}
	res.EnvVars = base.makeEnvVars(envVars)

	res.LoggingFields = map[string]string{
		"trial_id": strconv.Itoa(s.InitialWorkload.TrialID),
	}

	res.UseFluentLogging = true

	res.UseHostMode = s.IsMultiAgent

	if shm := s.ExperimentConfig.Resources().ShmSize(); shm != nil {
		res.ShmSize = int64(*shm)
	}
	res.ShmSize = 0

	res.ResourcesConfig = s.ExperimentConfig.Resources()

	mounts := ToDockerMounts(s.ExperimentConfig.BindMounts())
	addMount := func(source, target string, bindOpts *mount.BindOptions) {
		mounts = append(mounts, mount.Mount{
			Type: mount.TypeBind, Source: source, Target: target, BindOptions: bindOpts,
		})
	}
	if c := s.ExperimentConfig.CheckpointStorage().RawSharedFSConfig; c != nil {
		addMount(
			c.HostPath(),
			model.DefaultSharedFSContainerPath,
			&mount.BindOptions{Propagation: model.DefaultSharedFSPropagation},
		)
	}
	if c := s.ExperimentConfig.DataLayer().RawSharedFSConfig; c != nil {
		if c.HostStoragePath() != nil && c.ContainerStoragePath() != nil {
			addMount(*c.HostStoragePath(), *c.ContainerStoragePath(), nil)
		}
	}
	if c := s.ExperimentConfig.DataLayer().RawS3Config; c != nil {
		if c.LocalCacheHostPath() != nil && c.LocalCacheContainerPath() != nil {
			addMount(*c.LocalCacheHostPath(), *c.LocalCacheContainerPath(), nil)
		}
	}
	if c := s.ExperimentConfig.DataLayer().RawGCSConfig; c != nil {
		if c.LocalCacheHostPath() != nil && c.LocalCacheContainerPath() != nil {
			addMount(*c.LocalCacheHostPath(), *c.LocalCacheContainerPath(), nil)
		}
	}
	res.Mounts = mounts

	return res
}

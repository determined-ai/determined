package tasks

import (
	"archive/tar"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/ssh"
)

// TrialSpec is a description of a task for running a trial container.
type TrialSpec struct {
	Base TaskSpec

	ExperimentID     int
	TrialID          int
	TrialRunID       int
	ExperimentConfig expconf.ExperimentConfig
	HParams          map[string]interface{}
	TrialSeed        uint32
	LatestCheckpoint *model.Checkpoint
	LatestBatch      int
}

// ToTaskSpec generates a TaskSpec.
func (s TrialSpec) ToTaskSpec(keys *ssh.PrivateAndPublicKeys) TaskSpec {
	res := s.Base

	additionalFiles := archive.Archive{
		s.Base.AgentUserGroup.OwnedArchiveItem(
			trialEntrypointFile,
			etc.MustStaticFile(etc.TrialEntrypointScriptResource),
			trialEntrypointMode,
			tar.TypeReg,
		),

		s.Base.AgentUserGroup.OwnedArchiveItem(sshDir, nil, sshDirMode, tar.TypeDir),
		s.Base.AgentUserGroup.OwnedArchiveItem(trialAuthorizedKeysFile,
			keys.PublicKey,
			trialAuthorizedKeysMode,
			tar.TypeReg,
		),
		s.Base.AgentUserGroup.OwnedArchiveItem(
			pubKeyFile, keys.PublicKey, pubKeyMode, tar.TypeReg,
		),
		s.Base.AgentUserGroup.OwnedArchiveItem(
			privKeyFile, keys.PrivateKey, privKeyMode, tar.TypeReg,
		),
		s.Base.AgentUserGroup.OwnedArchiveItem(sshdConfigFile,
			etc.MustStaticFile(etc.SSHDConfigResource),
			sshdConfigMode,
			tar.TypeReg,
		),

		archive.RootItem(
			trialSSHConfigFile,
			etc.MustStaticFile(etc.SSHConfigResource),
			trialSSHConfigMode,
			tar.TypeReg,
		),
	}

	res.ExtraArchives = []container.RunArchive{
		wrapArchive(
			archive.Archive{
				s.Base.AgentUserGroup.OwnedArchiveItem(trainDir, nil, 0700, tar.TypeDir),
				s.Base.AgentUserGroup.OwnedArchiveItem(modelCopy, nil, 0700, tar.TypeDir),
			},
			rootDir,
		),
		wrapArchive(additionalFiles, rootDir),
		wrapArchive(
			archive.Archive{
				s.Base.AgentUserGroup.OwnedArchiveItem(
					"checkpoint.json",
					[]byte(jsonify(s.LatestCheckpoint)),
					0600,
					tar.TypeReg,
				),
			},
			trainDir,
		),
	}

	res.Description = fmt.Sprintf(
		"exp-%d-trial-%d",
		s.ExperimentID,
		s.TrialID,
	)

	res.Entrypoint = []string{"/run/determined/train/entrypoint.sh"}

	env := s.ExperimentConfig.Environment()
	ports := env.Ports()
	if ports == nil {
		ports = make(map[string]int)
	}
	ports["trial"] = rendezvousPort(trialUniquePortOffset(s.Base.Devices))
	env.SetPorts(ports)
	res.Environment = env

	portOffset := trialUniquePortOffset(s.Base.Devices)
	portStr := rendezvousPort(portOffset)
	envVars := map[string]string{
		"DET_EXPERIMENT_ID":            strconv.Itoa(s.ExperimentID),
		"DET_TRIAL_ID":                 strconv.Itoa(s.TrialID),
		"DET_TRIAL_RUN_ID":             strconv.Itoa(s.TrialRunID),
		"DET_TRIAL_SEED":               strconv.FormatUint(uint64(s.TrialSeed), 10),
		"DET_EXPERIMENT_CONFIG":        jsonify(s.ExperimentConfig),
		"DET_HPARAMS":                  jsonify(s.HParams),
		"DET_LATEST_BATCH":             strconv.Itoa(s.LatestBatch),
		"DET_LATEST_CHECKPOINT":        "/run/determined/train/checkpoint.json",
		"DET_RENDEZVOUS_PORT":          strconv.Itoa(portStr),
		"DET_TRIAL_UNIQUE_PORT_OFFSET": strconv.Itoa(portOffset),
	}
	res.ExtraEnvVars = envVars

	res.LoggingFields = map[string]string{
		"trial_id": strconv.Itoa(s.TrialID),
	}

	res.UseFluentLogging = true

	if shm := s.ExperimentConfig.Resources().ShmSize(); shm != nil {
		res.ShmSize = int64(*shm)
	}

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

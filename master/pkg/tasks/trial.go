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
	"github.com/determined-ai/determined/master/pkg/workload"
)

// StartTrial is a description of a task for running a trial container.
type StartTrial struct {
	ExperimentConfig    expconf.ExperimentConfig
	ModelDefinition     archive.Archive
	HParams             map[string]interface{}
	TrialSeed           uint32
	LatestCheckpoint    *model.Checkpoint
	InitialWorkload     workload.Workload
	WorkloadManagerType model.WorkloadManagerType

	PrivateKey []byte
	PublicKey  []byte

	// This is used to hint the resource manager to override defaults and start
	// the container in host mode iff it has been scheduled across multiple agents.
	IsMultiAgent bool

	Rank int
}

func (s StartTrial) additonalFiles(u *model.AgentUserGroup) archive.Archive {
	const (
		trialEntrypointFile = "/run/determined/train/entrypoint.sh"
		trialEntrypointMode = 0744

		// Put as many ssh-related files in /run/determined as possible. In particular, it is very
		// important that we don't overwrite the user's host $HOME/.ssh/id_rsa, if the user happens to
		// mount their host $HOME into the container's $HOME. Since we control the invocation of sshd,
		// we can keep our sshd_config in a location not likely to be mounted by users.
		trialAuthorizedKeysFile = "/run/determined/ssh/authorized_keys"
		trialAuthorizedKeysMode = 0600
		trialRSAPublicKeyFile   = "/run/determined/ssh/id_rsa.pub"
		trialRSAPublicKeyMode   = 0600
		trialRSAPrivateKeyFile  = "/run/determined/ssh/id_rsa"
		trialRSAPrivateKeyMode  = 0600
		trialSSHDConfigFile     = "/run/determined/ssh/sshd_config"
		trialSSHDConfigMode     = 0600
		trialSSHDir             = "/run/determined/ssh"
		trialSSHDirMode         = 0700

		// horovodrun controls how ssh is invoked, and we are force to overwrite a default ssh
		// configuration file.
		trialSSHConfigFile = "/etc/ssh/ssh_config"
		trialSSHConfigMode = 0644
	)

	return archive.Archive{
		u.OwnedArchiveItem(
			trialEntrypointFile,
			etc.MustStaticFile(etc.TrialEntrypointScriptResource),
			trialEntrypointMode,
			tar.TypeReg,
		),

		u.OwnedArchiveItem(trialSSHDir, nil, trialSSHDirMode, tar.TypeDir),
		u.OwnedArchiveItem(trialAuthorizedKeysFile,
			s.PublicKey,
			trialAuthorizedKeysMode,
			tar.TypeReg,
		),
		u.OwnedArchiveItem(
			trialRSAPublicKeyFile, s.PublicKey, trialRSAPublicKeyMode, tar.TypeReg,
		),
		u.OwnedArchiveItem(
			trialRSAPrivateKeyFile, s.PrivateKey, trialRSAPrivateKeyMode, tar.TypeReg,
		),
		u.OwnedArchiveItem(trialSSHDConfigFile,
			etc.MustStaticFile(etc.SSHDConfigResource),
			trialSSHDConfigMode,
			tar.TypeReg,
		),

		archive.RootItem(
			trialSSHConfigFile,
			etc.MustStaticFile(etc.SSHConfigResource),
			trialSSHConfigMode,
			tar.TypeReg,
		),
	}
}

// Archives implements InnerSpec.
func (s StartTrial) Archives(u *model.AgentUserGroup) []container.RunArchive {
	return []container.RunArchive{
		wrapArchive(
			archive.Archive{
				u.OwnedArchiveItem(trainDir, nil, 0700, tar.TypeDir),
				u.OwnedArchiveItem(modelCopy, nil, 0700, tar.TypeDir),
			},
			rootDir,
		),
		wrapArchive(s.additonalFiles(u), rootDir),
		wrapArchive(
			archive.Archive{
				u.OwnedArchiveItem(
					"checkpoint.json",
					[]byte(jsonify(s.LatestCheckpoint)),
					0600,
					tar.TypeReg,
				),
			},
			trainDir,
		),
		wrapArchive(u.OwnArchive(s.ModelDefinition), modelCopy),
		wrapArchive(u.OwnArchive(s.ModelDefinition), ContainerWorkDir),
	}
}

// Description implements InnerSpec.
func (s StartTrial) Description() string {
	return fmt.Sprintf(
		"exp-%d-trial-%d-rank-%d",
		s.InitialWorkload.ExperimentID,
		s.InitialWorkload.TrialID,
		s.Rank,
	)
}

// Entrypoint implements InnerSpec.
func (s StartTrial) Entrypoint() []string {
	return []string{"/run/determined/train/entrypoint.sh"}
}

// Environment implements InnerSpec.
func (s StartTrial) Environment(t TaskSpec) expconf.EnvironmentConfig {
	env := s.ExperimentConfig.Environment()
	ports := env.Ports()
	if ports == nil {
		ports = make(map[string]int)
	}
	ports["trial"] = rendezvousPort(trialUniquePortOffset(t.Devices))
	env.SetPorts(ports)
	return env
}

// EnvVars implements InnerSpec.
func (s StartTrial) EnvVars(t TaskSpec) map[string]string {
	portOffset := trialUniquePortOffset(t.Devices)
	portStr := rendezvousPort(portOffset)
	return map[string]string{
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
}

// LoggingFields implements InnerSpec.
func (s StartTrial) LoggingFields() map[string]string {
	return map[string]string{
		"trial_id": strconv.Itoa(s.InitialWorkload.TrialID),
	}
}

// Mounts implements InnerSpec.
func (s StartTrial) Mounts() []mount.Mount {
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

	return mounts
}

// UseFluentLogging implements InnerSpec.
func (s StartTrial) UseFluentLogging() bool { return true }

// UseHostMode implements InnerSpec.
func (s StartTrial) UseHostMode() bool { return s.IsMultiAgent }

// ShmSize implements InnerSpec.
func (s StartTrial) ShmSize() int64 {
	if shm := s.ExperimentConfig.Resources().ShmSize(); shm != nil {
		return int64(*shm)
	}
	return 0
}

// ResourcesConfig implements InnerSpec.
func (s StartTrial) ResourcesConfig() expconf.ResourcesConfig {
	return s.ExperimentConfig.Resources()
}

package tasks

import (
	"archive/tar"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/mount"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
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
	StepsCompleted   int

	Keys ssh.PrivateAndPublicKeys
}

// ToTaskSpec generates a TaskSpec.
func (s TrialSpec) ToTaskSpec() TaskSpec {
	res := s.Base

	res.Environment = s.MakeEnvPorts()

	res.UniqueExposedPortRequests = map[string]int{
		DTrainSSHPort:              DtrainSSHPortBase,
		InterTrainProcessCommPort1: InterTrainProcessCommPort1Base,
		InterTrainProcessCommPort2: InterTrainProcessCommPort2Base,
		C10DPort:                   C10DPortBase,
	}

	res.ResourcesConfig = s.ExperimentConfig.Resources()
	res.SlurmConfig = s.ExperimentConfig.SlurmConfig()
	res.PbsConfig = s.ExperimentConfig.PbsConfig()

	res.WorkDir = DefaultWorkDir

	additionalFiles := archive.Archive{
		s.Base.AgentUserGroup.OwnedArchiveItem(
			trialEntrypointFile,
			etc.MustStaticFile(etc.TrialEntrypointScriptResource),
			trialEntrypointMode,
			tar.TypeReg,
		),
	}

	additionalSSHFiles := archive.Archive{
		s.Base.AgentUserGroup.OwnedArchiveItem(sshDir, nil, sshDirMode, tar.TypeDir),
		s.Base.AgentUserGroup.OwnedArchiveItem(trialAuthorizedKeysFile,
			s.Keys.PublicKey,
			trialAuthorizedKeysMode,
			tar.TypeReg,
		),
		s.Base.AgentUserGroup.OwnedArchiveItem(
			pubKeyFile, s.Keys.PublicKey, pubKeyMode, tar.TypeReg,
		),
		s.Base.AgentUserGroup.OwnedArchiveItem(
			privKeyFile, s.Keys.PrivateKey, privKeyMode, tar.TypeReg,
		),
		s.Base.AgentUserGroup.OwnedArchiveItem(sshdConfigFile,
			etc.MustStaticFile(etc.SSHDConfigResource),
			sshdConfigMode,
			tar.TypeReg,
		),
	}

	additionalRootFiles := archive.Archive{
		archive.RootItem(
			trialSSHConfigFile,
			etc.MustStaticFile(etc.SSHConfigResource),
			trialSSHConfigMode,
			tar.TypeReg,
		),
	}

	res.ExtraArchives = []cproto.RunArchive{
		wrapArchive(
			archive.Archive{
				s.Base.AgentUserGroup.OwnedArchiveItem(trainDir, nil, 0o700, tar.TypeDir),
				s.Base.AgentUserGroup.OwnedArchiveItem(modelCopy, nil, 0o700, tar.TypeDir),
			},
			rootDir,
		),
		wrapArchive(additionalFiles, rootDir),
		wrapArchive(additionalRootFiles, rootDir),
		wrapArchive(additionalSSHFiles, rootDir),
	}

	res.Description = fmt.Sprintf(
		"exp-%d-trial-%d",
		s.ExperimentID,
		s.TrialID,
	)

	res.Entrypoint = []string{"/run/determined/train/entrypoint.sh"}

	envVars := map[string]string{
		"DET_EXPERIMENT_ID":     strconv.Itoa(s.ExperimentID),
		"DET_TRIAL_ID":          strconv.Itoa(s.TrialID),
		"DET_TRIAL_RUN_ID":      strconv.Itoa(s.TrialRunID),
		"DET_TRIAL_SEED":        strconv.FormatUint(uint64(s.TrialSeed), 10),
		"DET_EXPERIMENT_CONFIG": jsonify(s.ExperimentConfig),
		"DET_HPARAMS":           jsonify(s.HParams),
		"DET_STEPS_COMPLETED":   strconv.Itoa(s.StepsCompleted),
		"DET_TASK_TYPE":         string(model.TaskTypeTrial),
	}
	if s.LatestCheckpoint != nil && s.LatestCheckpoint.UUID != nil {
		envVars["DET_LATEST_CHECKPOINT"] = s.LatestCheckpoint.UUID.String()
	}

	res.ExtraEnvVars = envVars

	if shm := s.ExperimentConfig.Resources().ShmSize(); shm != nil {
		res.ShmSize = int64(*shm)
	}

	mounts := ToDockerMounts(s.ExperimentConfig.BindMounts(), res.WorkDir)
	addMount := func(source, target string, bindOpts *mount.BindOptions) {
		mounts = append(mounts, mount.Mount{
			Type: mount.TypeBind, Source: source, Target: target, BindOptions: bindOpts,
		})
	}
	if c := s.ExperimentConfig.CheckpointStorage().RawSharedFSConfig; c != nil {
		addMount(
			c.HostPath(),
			expconf.DefaultSharedFSContainerPath,
			&mount.BindOptions{Propagation: expconf.DefaultSharedFSPropagation},
		)
	}
	res.Mounts = mounts
	res.TaskType = model.TaskTypeTrial

	return res
}

// MakeEnvPorts fills in `Environment.Ports` i.e. exposed ports for container config.
func (s *TrialSpec) MakeEnvPorts() expconf.EnvironmentConfigV0 {
	ppc := s.ProxyPorts()
	ports := s.ExperimentConfig.Environment().Ports()
	if ports == nil {
		ports = map[string]int{}
	}

	for _, pp := range ppc {
		port := pp.ProxyPort()
		ports[strconv.Itoa(port)] = port
	}

	// TODO: remove this, but without breaking rendezvous api.
	ports["trial"] = 1734

	env := s.ExperimentConfig.Environment()
	env.SetPorts(ports)

	return env
}

// TrialSpecProxyPorts combines user-defined and system proxy configs.
// This static function is public because trial actor builds `TrialSpec` instances late.
func TrialSpecProxyPorts(
	taskSpec *TaskSpec,
	expConfig expconf.ExperimentConfigV0,
) expconf.ProxyPortsConfig {
	env := expConfig.Environment()
	epp := schemas.WithDefaults(taskSpec.ExtraProxyPorts)
	out := make(expconf.ProxyPortsConfig, 0, len(epp)+len(env.ProxyPorts()))

	for _, pp := range epp {
		out = append(out, pp)
	}

	for _, pp := range env.ProxyPorts() {
		out = append(out, pp)
	}

	return out
}

// ProxyPorts combines user-defined and system proxy configs.
func (s *TrialSpec) ProxyPorts() expconf.ProxyPortsConfig {
	return TrialSpecProxyPorts(&s.Base, s.ExperimentConfig)
}

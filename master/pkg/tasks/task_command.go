package tasks

import (
	"archive/tar"
	"encoding/json"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ssh"
)

// genericCommandSpecMetadata is GenericCommandSpec.Metadata.
type genericCommandSpecMetadata struct {
	PrivateKey    *string             `json:"privateKey"`
	PublicKey     *string             `json:"publicKey"`
	ExperimentIDs []int32             `json:"experiment_ids"`
	TrialIDs      []int32             `json:"trial_ids"`
	WorkspaceID   model.AccessScopeID `json:"workspace_id"`
}

// MarshalToMap converts typed struct into a map.
func (metadata *genericCommandSpecMetadata) MarshalToMap() (map[string]interface{}, error) {
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})

	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GenericCommandSpec is a description of a task for running a command.
type GenericCommandSpec struct {
	Base TaskSpec

	Config          model.CommandConfig
	UserFiles       archive.Archive
	AdditionalFiles archive.Archive
	Metadata        genericCommandSpecMetadata

	Keys *ssh.PrivateAndPublicKeys

	Port            *int
	ProxyTCP        bool
	Unauthenticated bool

	WatchProxyIdleTimeout  bool
	WatchRunnerIdleTimeout bool

	TaskType model.TaskType
}

// ToTaskSpec generates a TaskSpec.
func (s GenericCommandSpec) ToTaskSpec(keys *ssh.PrivateAndPublicKeys) TaskSpec {
	res := s.Base

	res.Environment = s.Config.Environment.ToExpconf()

	res.ResourcesConfig = s.Config.Resources.ToExpconf()

	res.PbsConfig = s.Config.Pbs

	res.SlurmConfig = s.Config.Slurm

	res.WorkDir = DefaultWorkDir
	if s.Config.WorkDir != nil {
		res.WorkDir = *s.Config.WorkDir
	}
	res.ResolveWorkDir()

	if keys != nil {
		s.AdditionalFiles = append(s.AdditionalFiles, archive.Archive{
			res.AgentUserGroup.OwnedArchiveItem(sshDir, nil, sshDirMode, tar.TypeDir),
			res.AgentUserGroup.OwnedArchiveItem(
				shellAuthorizedKeysFile, keys.PublicKey, 0o644, tar.TypeReg,
			),
			res.AgentUserGroup.OwnedArchiveItem(
				privKeyFile, keys.PrivateKey, privKeyMode, tar.TypeReg,
			),
			res.AgentUserGroup.OwnedArchiveItem(
				pubKeyFile, keys.PublicKey, pubKeyMode, tar.TypeReg,
			),
			res.AgentUserGroup.OwnedArchiveItem(
				sshdConfigFile,
				etc.MustStaticFile(etc.SSHDConfigResource),
				0o644,
				tar.TypeReg,
			),
		}...)
	}

	res.ExtraArchives = []cproto.RunArchive{
		wrapArchive(s.Base.AgentUserGroup.OwnArchive(s.UserFiles), res.WorkDir),
		wrapArchive(s.AdditionalFiles, rootDir),
	}

	res.Description = "cmd"

	res.Entrypoint = s.Config.Entrypoint

	res.Mounts = ToDockerMounts(s.Config.BindMounts.ToExpconf(), res.WorkDir)

	if shm := s.Config.Resources.ShmSize; shm != nil {
		res.ShmSize = int64(*shm)
	}

	res.TaskType = s.TaskType
	return res
}

package tasks

import (
	"archive/tar"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ssh"
)

// GenericCommandSpec is a description of a task for running a command.
type GenericCommandSpec struct {
	Base TaskSpec

	Config          model.CommandConfig
	UserFiles       archive.Archive
	AdditionalFiles archive.Archive
	Metadata        map[string]interface{}

	Keys *ssh.PrivateAndPublicKeys

	Port     *int
	ProxyTCP bool

	WatchProxyIdleTimeout  bool
	WatchRunnerIdleTimeout bool
}

// ToTaskSpec generates a TaskSpec.
func (s GenericCommandSpec) ToTaskSpec(
	keys *ssh.PrivateAndPublicKeys,
) TaskSpec {
	res := s.Base

	res.Environment = s.Config.Environment.ToExpconf()

	res.ResourcesConfig = s.Config.Resources.ToExpconf()

	res.WorkDir = DefaultWorkDir
	if s.Config.WorkDir != nil {
		res.WorkDir = *s.Config.WorkDir
	}
	res.ResolveWorkDir()

	if keys != nil {
		s.AdditionalFiles = append(s.AdditionalFiles, archive.Archive{
			res.AgentUserGroup.OwnedArchiveItem(sshDir, nil, sshDirMode, tar.TypeDir),
			res.AgentUserGroup.OwnedArchiveItem(
				shellAuthorizedKeysFile, keys.PublicKey, 0644, tar.TypeReg,
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
				0644,
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

	res.UseFluentLogging = true

	return res
}

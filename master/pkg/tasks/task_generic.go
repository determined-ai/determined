package tasks

import (
	"archive/tar"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// GenericTaskSpec is the generic task spec.
type GenericTaskSpec struct {
	Base      TaskSpec
	ProjectID int

	GenericTaskConfig model.GenericTaskConfig
}

func (s GenericTaskSpec) ToTaskSpec() TaskSpec {
	res := s.Base

	commandEntrypoint := "/run/determined/generic-task-entrypoint.sh"
	res.Entrypoint = []string{commandEntrypoint}
	res.Entrypoint = append(res.Entrypoint, s.GenericTaskConfig.Entrypoint...)
	commandEntryArchive := wrapArchive(archive.Archive{
		res.AgentUserGroup.OwnedArchiveItem(
			commandEntrypoint,
			etc.MustStaticFile("generic-task-entrypoint.sh"),
			0o700,
			tar.TypeReg,
		),
	}, "/")

	// TODO proxy ports

	// TODO
	// res.PbsConfig = s.Config.Pbs
	// res.SlurmConfig = s.Config.Slurm

	res.ExtraArchives = []cproto.RunArchive{commandEntryArchive}
	res.TaskType = s.Base.TaskType
	res.Environment = s.GenericTaskConfig.Environment.ToExpconf()

	res.WorkDir = DefaultWorkDir
	if s.GenericTaskConfig.WorkDir != nil {
		res.WorkDir = *s.GenericTaskConfig.WorkDir
	}
	res.ResolveWorkDir()

	res.ResourcesConfig = s.GenericTaskConfig.Resources

	res.Description = "generic-task"

	fmt.Println("bind mounts", s.GenericTaskConfig.BindMounts)
	res.Mounts = ToDockerMounts(s.GenericTaskConfig.BindMounts.ToExpconf(), res.WorkDir)

	if shm := s.GenericTaskConfig.Resources.ShmSize(); shm != nil {
		res.ShmSize = int64(*shm)
	}

	res.TaskType = model.TaskTypeGeneric

	return res
}

// TODO
func (s GenericTaskSpec) ToV1Job() (*jobv1.Job, error)              { return nil, nil }
func (s GenericTaskSpec) SetJobPriority(priority int) error         { return nil }
func (s GenericTaskSpec) SetWeight(weight float64) error            { return nil }
func (s GenericTaskSpec) SetResourcePool(resourcePool string) error { return nil }

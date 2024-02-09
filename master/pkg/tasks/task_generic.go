package tasks

import (
	"archive/tar"

	"google.golang.org/protobuf/types/known/timestamppb"

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

// ToTaskSpec converts the generic task spec to the common task spec.
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

	// TODO proxy ports eventually.
	res.PbsConfig = s.GenericTaskConfig.Pbs
	res.SlurmConfig = s.GenericTaskConfig.Slurm

	res.ExtraArchives = []cproto.RunArchive{commandEntryArchive}
	res.Environment = s.GenericTaskConfig.Environment.ToExpconf()

	res.WorkDir = DefaultWorkDir
	if s.GenericTaskConfig.WorkDir != nil {
		res.WorkDir = *s.GenericTaskConfig.WorkDir
	}
	res.ResolveWorkDir()

	res.ResourcesConfig = s.GenericTaskConfig.Resources

	res.Description = "generic-task"

	res.Mounts = ToDockerMounts(s.GenericTaskConfig.BindMounts.ToExpconf(), res.WorkDir)

	if shm := s.GenericTaskConfig.Resources.ShmSize(); shm != nil {
		res.ShmSize = int64(*shm)
	}

	res.TaskType = model.TaskTypeGeneric

	return res
}

// TODO(aaron.amanuel): fill in job information. These should probably be on a different struct.
// not right on the generic task spec.

// ToV1Job todo.
func (s GenericTaskSpec) ToV1Job() (*jobv1.Job, error) {
	j := jobv1.Job{
		JobId:          "GENERIC TASK",
		EntityId:       s.Base.TaskID,
		Type:           jobv1.Type_TYPE_UNSPECIFIED,
		SubmissionTime: timestamppb.Now(),
		Username:       s.Base.Owner.Username,
		UserId:         0,
		Weight:         0,
		Name:           "ToV1Job not implemented for Generic Task",
		WorkspaceId:    0,
	}

	j.Priority = 0

	j.ResourcePool = s.ResourcePool()
	return &j, nil
}

// SetJobPriority todo.
func (s GenericTaskSpec) SetJobPriority(priority int) error { return nil }

// SetWeight todo.
func (s GenericTaskSpec) SetWeight(weight float64) error { return nil }

// SetResourcePool todo.
func (s GenericTaskSpec) SetResourcePool(resourcePool string) error { return nil }

// ResourcePool - returns resource pool.
func (s GenericTaskSpec) ResourcePool() string { return s.GenericTaskConfig.Resources.ResourcePool() }

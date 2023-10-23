package command

import (
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// job.Service methods

// ToV1Job() takes a command and returns a job.
func (c *command) ToV1Job() *jobv1.Job {
	c.mu.Lock()
	defer c.mu.Unlock()

	j := jobv1.Job{
		JobId:          c.jobID.String(),
		EntityId:       string(c.taskID),
		Type:           c.jobType.Proto(),
		SubmissionTime: timestamppb.New(c.registeredTime),
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		Weight:         c.Config.Resources.Weight,
		Name:           c.Config.Description,
		WorkspaceId:    int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}

	j.IsPreemptible = false
	j.Priority = int32(config.ReadPriority(j.ResourcePool, &c.Config))
	j.Weight = config.ReadWeight(j.ResourcePool, &c.Config)

	j.ResourcePool = c.Config.Resources.ResourcePool

	return &j
}

// SetJobPriority sets a command's job priority.
func (c *command) SetJobPriority(priority int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if priority < 1 || priority > 99 {
		return errors.New("priority must be between 1 and 99")
	}
	err := c.setPriority(priority, true)
	if err != nil {
		c.syslog.WithError(err).Info("setting command job priority")
	}
	return err
}

// SetWeight sets the command's group weight.
func (c *command) SetWeight(weight float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch err := c.rm.SetGroupWeight(sproto.SetGroupWeight{
		Weight: weight,
		JobID:  c.jobID,
	}).(type) {
	case nil:
	case rmerrors.ErrUnsupported:
		c.syslog.WithError(err).Debug("ignoring unsupported call to set group weight")
	default:
		return fmt.Errorf("setting group weight for command: %w", err)
	}

	c.Config.Resources.Weight = weight
	return nil
}

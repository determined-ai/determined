package command

import (
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/configpolicy"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// jobservice.Service methods

// ToV1Job takes a command and returns a job.
func (c *Command) ToV1Job() (*jobv1.Job, error) {
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

	return &j, nil
}

// SetJobPriority sets a command's job priority.
func (c *Command) SetJobPriority(priority int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if priority < 1 || priority > 99 {
		return fmt.Errorf("priority must be between 1 and 99")
	}

	// Returns an error if RM does not implement priority.
	if smallerHigher, err := c.rm.SmallerValueIsHigherPriority(); err == nil {
		ok, err := configpolicy.PriorityAllowed(
			int(c.GenericCommandSpec.Metadata.WorkspaceID),
			model.NTSCType,
			priority,
			smallerHigher,
		)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("priority exceeds task config policy's priority_limit")
		}
	}

	err := c.setNTSCPriority(priority, true)
	if err != nil {
		c.syslog.WithError(err).Info("setting command job priority")
	}
	return err
}

// SetWeight sets the command's group weight.
func (c *Command) SetWeight(weight float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch err := c.rm.SetGroupWeight(sproto.SetGroupWeight{
		Weight:       weight,
		ResourcePool: c.Config.Resources.ResourcePool,
		JobID:        c.jobID,
	}).(type) {
	case nil:
	case rmerrors.UnsupportedError:
		c.syslog.WithError(err).Debug("ignoring unsupported call to set group weight")
	default:
		return fmt.Errorf("setting group weight for command: %w", err)
	}

	c.Config.Resources.Weight = weight
	return nil
}

// SetResourcePool is not implemented for commands.
func (c *Command) SetResourcePool(resourcePool string) error {
	return fmt.Errorf("setting resource pool for job type %s is not supported", c.jobType)
}

// ResourcePool gets the command's resource pool.
func (c *Command) ResourcePool() string {
	return c.Config.Resources.ResourcePool
}

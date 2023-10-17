package internal

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// job.Service methods

// ToV1Jobs() takes an experiment and returns a job.
func (e *experiment) ToV1Job() *jobv1.Job {
	e.mu.Lock()
	defer e.mu.Unlock()

	workspace, err := workspace.WorkspaceByProjectID(context.TODO(), e.ProjectID)
	if err != nil && err != sql.ErrNoRows {
		// FIXME: DET-9563 workspace and/or project is deleted.
		e.syslog.WithError(err)
		return nil
	}

	j := jobv1.Job{
		JobId:          e.JobID.String(),
		EntityId:       fmt.Sprint(e.ID),
		Type:           jobv1.Type_TYPE_EXPERIMENT,
		SubmissionTime: timestamppb.New(e.StartTime),
		Username:       e.Username,
		UserId:         int32(*e.OwnerID),
		Progress:       float32(e.searcher.Progress()),
		Name:           e.activeConfig.Name().String(),
		WorkspaceId:    int32(workspace.ID),
	}

	j.IsPreemptible = config.ReadRMPreemptionStatus(j.ResourcePool)
	j.Priority = int32(config.ReadPriority(j.ResourcePool, &e.activeConfig))
	j.Weight = config.ReadWeight(j.ResourcePool, &e.activeConfig)

	j.ResourcePool = e.activeConfig.Resources().ResourcePool()

	return &j
}

// SetJobPriority sets an experiment's job priority.
func (e *experiment) SetJobPriority(priority int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if priority < 1 || priority > 99 {
		return errors.New("priority must be between 1 and 99")
	}
	err := e.setPriority(&priority, true)
	if err != nil {
		e.syslog.WithError(err).Info("setting experiment job priority")
	}
	return err
}

// SetWeight sets the experiment's group weight.
func (e *experiment) SetWeight(weight float64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	err := e.setWeight(weight)
	if err != nil {
		e.syslog.WithError(err).Info("setting experiment job weight")
	}
	return err
}

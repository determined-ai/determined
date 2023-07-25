//go:build integration

package internal

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

func TestRunCheckpointGCTask(t *testing.T) {
	system := actor.NewSystem(uuid.NewString())

	var rm mocks.ResourceManager
	pgDB := db.MustSetupTestPostgres(t)

	type args struct {
		taskID              model.TaskID
		jobID               model.JobID
		taskSpec            tasks.TaskSpec
		expID               int
		legacyConfig        expconf.LegacyConfig
		toDeleteCheckpoints []uuid.UUID
		checkpointGlobs     []string
		deleteTensorboards  bool
		agentUserGroup      *model.AgentUserGroup
		owner               *model.User
		logCtx              logger.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "successful task",
			args: args{
				taskID: model.NewTaskID(),
				jobID:  model.NewJobID(),
				// taskSpec:            tasks.TaskSpec{},
				// expID:               0,
				// legacyConfig:        expconf.LegacyConfig{},
				// toDeleteCheckpoints: []uuid.UUID{},
				// checkpointGlobs:     []string{},
				// deleteTensorboards:  false,
				// agentUserGroup:      &model.AgentUserGroup{},
				// owner:               &model.User{},
				// logCtx:              map[string]interface{}{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runCheckpointGCTask(
				system,
				&rm,
				pgDB,
				tt.args.taskID,
				tt.args.jobID,
				time.Now(),
				tasks.TaskSpec{},
				tt.args.expID,
				tt.args.legacyConfig,
				tt.args.toDeleteCheckpoints,
				tt.args.checkpointGlobs,
				tt.args.deleteTensorboards,
				tt.args.agentUserGroup,
				tt.args.owner,
				tt.args.logCtx,
			); (err != nil) != tt.wantErr {
				t.Errorf("runCheckpointGCTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

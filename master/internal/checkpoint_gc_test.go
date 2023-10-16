//go:build integration

package internal

import (
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/mocks/allocationmocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

func TestRunCheckpointGCTask(t *testing.T) {
	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, "file://../static/migrations")
	user := db.RequireMockUser(t, pgDB)

	type args struct {
		rm                  *mocks.ResourceManager
		as                  func(t *testing.T) *allocationmocks.AllocationService
		toDeleteCheckpoints []uuid.UUID
		checkpointGlobs     []string
		deleteTensorboards  bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "delete nothing does nothing",
			args: args{
				rm: func() *mocks.ResourceManager {
					return &mocks.ResourceManager{}
				}(),
				as: func(t *testing.T) *allocationmocks.AllocationService {
					return &allocationmocks.AllocationService{}
				},
			},
			wantErr: false,
		},
		{
			name: "simple success",
			args: args{
				rm: func() *mocks.ResourceManager {
					var rm mocks.ResourceManager

					rm.On("ResolveResourcePool", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
						Return("default", nil)

					rm.On("TaskContainerDefaults", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
						Return(model.TaskContainerDefaultsConfig{}, nil)

					return &rm
				}(),
				as: func(t *testing.T) *allocationmocks.AllocationService {
					var as allocationmocks.AllocationService

					as.On(
						"StartAllocation",
						mock.Anything,
						mock.MatchedBy(func(ar sproto.AllocateRequest) bool {
							return ar.IsUserVisible == false &&
								ar.ResourcePool == "default" &&
								ar.SlotsNeeded == 0
						}),
						mock.Anything,
						mock.Anything,
						mock.MatchedBy(func(spec tasks.GCCkptSpec) bool {
							ok := true
							if spec.ToDelete == "" {
								t.Error("to delete was not set")
								ok = false
							}
							if !spec.DeleteTensorboards {
								t.Error("delete tensorboards was not set")
								ok = false
							}
							if spec.CheckpointGlobs == nil {
								t.Error("checkpoint globs missing")
								ok = false
							}
							return ok
						}),
						mock.Anything,
					).Return(nil).Run(func(args mock.Arguments) {
						cb := args.Get(5).(func(*task.AllocationExited))
						cb(&task.AllocationExited{FinalState: task.AllocationState{
							State: model.AllocationStateTerminated,
						}})
					})

					return &as
				},
				toDeleteCheckpoints: []uuid.UUID{uuid.New()},
				checkpointGlobs:     []string{"optimizer_state.pkl"},
				deleteTensorboards:  true,
			},
			wantErr: false,
		},
		{
			name: "simple failure",
			args: args{
				rm: func() *mocks.ResourceManager {
					var rm mocks.ResourceManager

					rm.On("ResolveResourcePool", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
						Return("", errors.New("rm is down or something"))

					return &rm
				}(),
				as: func(t *testing.T) *allocationmocks.AllocationService {
					return &allocationmocks.AllocationService{}
				},
				toDeleteCheckpoints: []uuid.UUID{uuid.New()},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := task.DefaultService
			task.DefaultService = tt.args.as(t)
			defer func() { task.DefaultService = tmp }()

			jobID := db.RequireMockJob(t, pgDB, &user.ID)

			if err := runCheckpointGCTask(
				tt.args.rm,
				pgDB,
				model.NewTaskID(),
				jobID,
				time.Now(),
				tasks.TaskSpec{},
				0,
				expconf.LegacyConfig{}, //nolint:exhaustivestruct
				tt.args.toDeleteCheckpoints,
				tt.args.checkpointGlobs,
				tt.args.deleteTensorboards,
				nil,
				&user,
				nil,
			); (err != nil) != tt.wantErr {
				t.Errorf("runCheckpointGCTask() error = %v, wantErr %v", err, tt.wantErr)
			}

			require.True(t, tt.args.rm.AssertExpectations(t))
		})
	}
}

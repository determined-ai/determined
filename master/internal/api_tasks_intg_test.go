//go:build integration
// +build integration

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/logpattern"
	"github.com/determined-ai/determined/master/internal/sproto"
	taskPkg "github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/logv1"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

func TestPostTaskLogs(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, task := createTestTrial(t, api, curUser)
	_, task2 := createTestTrial(t, api, curUser)

	_, err := api.PostTaskLogs(ctx, &apiv1.PostTaskLogsRequest{})
	require.ErrorContains(t, err, "greater than 0")

	_, err = api.PostTaskLogs(ctx, &apiv1.PostTaskLogsRequest{
		Logs: []*taskv1.TaskLog{
			{TaskId: string(task.TaskID), Id: ptrs.Ptr(int32(2))},
		},
	})
	require.ErrorContains(t, err, "ID must be nil")

	_, err = api.PostTaskLogs(ctx, &apiv1.PostTaskLogsRequest{
		Logs: []*taskv1.TaskLog{
			{TaskId: string(task.TaskID)},
			{TaskId: string(task2.TaskID)},
		},
	})
	require.ErrorContains(t, err, "single taskID per task log")

	expected := []*taskv1.TaskLog{
		{
			TaskId: string(task.TaskID),
			Log:    "test",
		},
		{
			TaskId:       string(task.TaskID),
			AllocationId: ptrs.Ptr("alloc_id"),
			AgentId:      ptrs.Ptr("agent_id"),
			ContainerId:  ptrs.Ptr("container_id"),
			RankId:       ptrs.Ptr(int32(9)),
			Timestamp:    timestamppb.New(time.Now().Truncate(time.Millisecond)),
			Level:        ptrs.Ptr(logv1.LogLevel_LOG_LEVEL_WARNING),
			Log:          "log_text",
			Source:       ptrs.Ptr("source"),
			Stdtype:      ptrs.Ptr("stderr"),
		},
	}

	_, err = api.PostTaskLogs(ctx, &apiv1.PostTaskLogsRequest{
		Logs: expected,
	})
	require.NoError(t, err)

	stream := &mockStream[*apiv1.TaskLogsResponse]{ctx: ctx}
	err = api.TaskLogs(&apiv1.TaskLogsRequest{
		TaskId: string(task.TaskID),
	}, stream)
	require.NoError(t, err)

	require.Len(t, stream.getData(), len(expected))
	for i, a := range stream.getData() {
		e := expected[i]

		require.NotEmpty(t, a.Id)
		require.Equal(t, e.Timestamp.AsTime(), a.Timestamp.AsTime())
		require.NotEmpty(t, a.Message) //nolint: staticcheck
		require.Equal(t, e.TaskId, a.TaskId)
		require.Equal(t, e.AllocationId, a.AllocationId)
		require.Equal(t, e.AgentId, a.AgentId)
		require.Equal(t, e.ContainerId, a.ContainerId)
		require.Equal(t, e.RankId, a.RankId)
		require.Equal(t, e.Log, a.Log)
		require.Equal(t, e.Source, a.Source)
		require.Equal(t, e.Stdtype, a.Stdtype)
	}

	// Test log filtering by regex
	stream = &mockStream[*apiv1.TaskLogsResponse]{ctx: ctx}
	err = api.TaskLogs(&apiv1.TaskLogsRequest{
		TaskId:     string(task.TaskID),
		SearchText: "^lo.{4}xt",
	}, stream)
	require.NoError(t, err)
	require.Empty(t, stream.getData())

	err = api.TaskLogs(&apiv1.TaskLogsRequest{
		TaskId:      string(task.TaskID),
		SearchText:  "^lo.{4}xt",
		EnableRegex: true,
	}, stream)
	require.NoError(t, err)
	require.Len(t, stream.getData(), 1)
}

func mockNotebookWithWorkspaceID(
	ctx context.Context, t *testing.T, workspaceID int,
) model.TaskID {
	nb := &model.Task{
		TaskID:   model.NewTaskID(),
		TaskType: model.TaskTypeNotebook,
	}
	require.NoError(t, db.AddTask(ctx, nb))

	allocationID := model.AllocationID(string(nb.TaskID) + ".1")
	require.NoError(t, db.AddAllocation(ctx, &model.Allocation{
		TaskID:       nb.TaskID,
		AllocationID: allocationID,
	}))

	type commandSnapshot struct { // can't use command.CommandSnapshot since metadata isn't exposed.
		bun.BaseModel `bun:"table:command_state"`

		TaskID             model.TaskID       `bun:"task_id"`
		RegisteredTime     time.Time          `bun:"registered_time"`
		AllocationID       model.AllocationID `bun:"allocation_id"`
		GenericCommandSpec map[string]any     `bun:"generic_command_spec"`
	}

	_, err := db.Bun().NewInsert().Model(&commandSnapshot{
		TaskID:       nb.TaskID,
		AllocationID: allocationID,
		GenericCommandSpec: map[string]any{
			"Metadata": map[string]any{
				"workspace_id": workspaceID,
			},
		},
	}).Exec(ctx)
	require.NoError(t, err)

	return nb.TaskID
}

func TestGetTasksAuthZ(t *testing.T) {
	var allocations map[model.AllocationID]sproto.AllocationSummary

	mockRM := MockRM()
	mockRM.On("GetAllocationSummaries", mock.Anything).Return(func() map[model.AllocationID]sproto.AllocationSummary {
		return allocations
	}, nil)

	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil, mockRM)
	_, authZNSC, _, _ := setupNTSCAuthzTest(t) //nolint: dogsled

	canAccessTrial, expCanAccessTask := createTestTrial(t, api, curUser)
	authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.MatchedBy(
		func(e *model.Experiment) bool {
			return e.ID == canAccessTrial.ExperimentID
		})).Return(nil).Once()

	cantAccessTrial, expCantAccessTask := createTestTrial(t, api, curUser)
	authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.MatchedBy(
		func(e *model.Experiment) bool {
			return e.ID == cantAccessTrial.ExperimentID
		})).Return(authz2.PermissionDeniedError{}).Once()

	canAccessNotebookID := mockNotebookWithWorkspaceID(ctx, t, -100)
	authZNSC.On("CanGetNSC", mock.Anything, mock.Anything, model.AccessScopeID(-100)).
		Return(nil).Once()

	cantAccessNotebookID := mockNotebookWithWorkspaceID(ctx, t, -101)
	authZNSC.On("CanGetNSC", mock.Anything, mock.Anything, model.AccessScopeID(-101)).
		Return(authz2.PermissionDeniedError{}).Once()

	allocations = map[model.AllocationID]sproto.AllocationSummary{
		"alloc0": {
			TaskID: expCanAccessTask.TaskID,
		},
		"alloc1": {
			TaskID: expCantAccessTask.TaskID,
		},
		"alloc2": {
			TaskID: canAccessNotebookID,
		},
		"alloc3": {
			TaskID: cantAccessNotebookID,
		},
	}

	resp, err := api.GetTasks(ctx, &apiv1.GetTasksRequest{})
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"alloc0", "alloc2"}, maps.Keys(resp.AllocationIdToSummary))
}

func TestPostTaskLogsLogPattern(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	trial, task := createTestTrial(t, api, curUser)

	activeConfig, err := api.m.db.ActiveExperimentConfig(trial.ExperimentID)
	require.NoError(t, err)
	activeConfig.RawLogPolicies = &expconf.LogPoliciesConfig{
		expconf.LogPolicy{RawPattern: "sub", RawActions: []expconf.LogActionV0{{
			RawCancelRetries: &expconf.LogActionCancelRetries{},
		}}},
		expconf.LogPolicy{RawPattern: `\d{5}$`, RawActions: []expconf.LogActionV0{{
			RawExcludeNode: &expconf.LogActionExcludeNode{},
		}}},
	}

	v, err := json.Marshal(activeConfig)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(v, &m))

	_, err = db.Bun().NewUpdate().Table("experiments").
		Where("id = ?", trial.ExperimentID).
		Set("config = ?", m).
		Exec(ctx)
	require.NoError(t, err)

	_, err = api.PostTaskLogs(ctx, &apiv1.PostTaskLogsRequest{
		Logs: []*taskv1.TaskLog{
			{
				TaskId:  string(task.TaskID),
				AgentId: ptrs.Ptr("a1"),
				Log:     "stringsubstring",
			},
			{
				TaskId:  string(task.TaskID),
				AgentId: ptrs.Ptr("a1"),
				Log:     "12345",
			},
		},
	})
	require.NoError(t, err)

	disallowed, err := logpattern.GetBlockedNodes(ctx, task.TaskID)
	require.NoError(t, err)
	require.Equal(t, []string{"a1"}, disallowed)

	retryInfo, err := logpattern.ShouldRetry(ctx, task.TaskID)
	require.NoError(t, err)
	require.Equal(t,
		[]logpattern.DontRetryTrigger{{Regex: `sub`, TriggeringLog: "stringsubstring"}},
		retryInfo)
}

func TestTaskAuthZ(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)

	mockUserArg := mock.MatchedBy(func(u model.User) bool {
		return u.ID == curUser.ID
	})

	_, task := createTestTrial(t, api, curUser)
	taskID := string(task.TaskID)

	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id string) error
	}{
		{"CanGetExperimentArtifacts", func(id string) error {
			_, err := api.GetTask(ctx, &apiv1.GetTaskRequest{
				TaskId: id,
			})
			return err
		}},
		{"CanEditExperiment", func(id string) error {
			_, err := api.ReportCheckpoint(ctx, &apiv1.ReportCheckpointRequest{
				Checkpoint: &checkpointv1.Checkpoint{TaskId: id},
			})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id string) error {
			return api.TaskLogs(&apiv1.TaskLogsRequest{
				TaskId: id,
			}, &mockStream[*apiv1.TaskLogsResponse]{ctx: ctx})
		}},
		{"CanGetExperimentArtifacts", func(id string) error {
			return api.TaskLogsFields(&apiv1.TaskLogsFieldsRequest{
				TaskId: id,
			}, &mockStream[*apiv1.TaskLogsFieldsResponse]{ctx: ctx})
		}},
	}

	for _, curCase := range cases {
		require.ErrorIs(t, curCase.IDToReqCall("-999"), apiPkg.NotFoundErrs("task", "-999", true))

		// Can't view allocation's experiment gives same error.
		authZExp.On("CanGetExperiment", mock.Anything, mockUserArg, mock.Anything).
			Return(authz2.PermissionDeniedError{}).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), apiPkg.NotFoundErrs("task", taskID, true))

		// Experiment view error is returned unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("CanGetExperiment", mock.Anything, mockUserArg, mock.Anything).
			Return(expectedErr).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), expectedErr)

		// Action func error returns err in forbidden.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(nil).Once()
		authZExp.On(curCase.DenyFuncName, mock.Anything, mockUserArg, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), expectedErr)
	}
}

// Checks if AddAllocationAcceleratorData and GetAllocationAcceleratorData work.
func TestAddAllocationAcceleratorData(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	tID := model.NewTaskID()
	task := &model.Task{
		TaskID:    tID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}
	require.NoError(t, db.AddTask(ctx, task), "failed to add task")

	aID := tID + "-1"
	a := &model.Allocation{
		TaskID:       tID,
		AllocationID: model.AllocationID(aID),
		Slots:        1,
		ResourcePool: "default",
	}
	require.NoError(t, db.AddAllocation(ctx, a), "failed to add allocation")
	accData := &model.AcceleratorData{
		ContainerID:      uuid.NewString(),
		AllocationID:     model.AllocationID(aID),
		NodeName:         "NodeName",
		AcceleratorType:  "cpu-test",
		AcceleratorUuids: []string{"g", "h", "i"},
	}
	require.NoError(t,
		taskPkg.AddAllocationAcceleratorData(ctx, *accData), "failed to add allocation")

	resp, err := api.GetTaskAcceleratorData(ctx,
		&apiv1.GetTaskAcceleratorDataRequest{TaskId: tID.String()})
	require.NoError(t, err, "failed to get task AccelerationData")
	require.Len(t, resp.AcceleratorData, 1, "incorrect number of allocation accelerator data returned")
	require.Equal(t, resp.AcceleratorData[0].AllocationId,
		aID.String(), "failed to get the correct allocation's accelerator data")
}

// Checks if GetAllocationAcceleratorData works when a task has only one allocation and it does
// not have accelerator data.
func TestGetAllocationAcceleratorDataWithNoData(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	tID := model.NewTaskID()
	task := &model.Task{
		TaskID:    tID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}
	require.NoError(t, db.AddTask(ctx, task), "failed to add task")

	aID := tID + "-1"
	a := &model.Allocation{
		TaskID:       tID,
		AllocationID: model.AllocationID(aID),
		Slots:        1,
		ResourcePool: "default",
	}
	require.NoError(t, db.AddAllocation(ctx, a), "failed to add allocation")

	resp, err := api.GetTaskAcceleratorData(ctx,
		&apiv1.GetTaskAcceleratorDataRequest{TaskId: tID.String()})
	require.NoError(t, err, "failed to get task AccelerationData")
	require.Empty(t, resp.AcceleratorData, "unexpected allocation accelerator data returned")
}

// Checks if GetAllocationAcceleratorData works when a task has more than one allocation
// but one without accelerator data.
func TestGetAllocationAcceleratorData(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	tID := model.NewTaskID()
	task := &model.Task{
		TaskID:    tID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}
	require.NoError(t, db.AddTask(ctx, task), "failed to add task")

	aID1 := tID + "-1"
	a1 := &model.Allocation{
		TaskID:       tID,
		AllocationID: model.AllocationID(aID1),
		Slots:        1,
		ResourcePool: "default",
	}
	require.NoError(t, db.AddAllocation(ctx, a1), "failed to add allocation")
	accData := &model.AcceleratorData{
		ContainerID:      uuid.NewString(),
		AllocationID:     model.AllocationID(aID1),
		NodeName:         "NodeName",
		AcceleratorType:  "cpu-test",
		AcceleratorUuids: []string{"a", "b", "c"},
	}
	require.NoError(t,
		taskPkg.AddAllocationAcceleratorData(ctx, *accData), "failed to add allocation")

	// Add another allocation that does not have associated acceleration data with it.
	aID2 := tID + "-2"
	a2 := &model.Allocation{
		TaskID:       tID,
		AllocationID: model.AllocationID(aID2),
		Slots:        1,
		ResourcePool: "default",
	}
	require.NoError(t, db.AddAllocation(ctx, a2), "failed to add allocation")

	resp, err := api.GetTaskAcceleratorData(ctx,
		&apiv1.GetTaskAcceleratorDataRequest{TaskId: tID.String()})
	require.NoError(t, err, "failed to get task AccelerationData")
	require.Len(t, resp.AcceleratorData, 1, "incorrect number of allocation accelerator data returned")
	require.Equal(t, resp.AcceleratorData[0].AllocationId,
		aID1.String(), "failed to get the correct allocation's accelerator data")
	require.Equal(t, resp.AcceleratorData[0].ResourcePool,
		a1.ResourcePool, "failed to get the correct allocation's resource pool data")
}

func TestPostTaskLogsLogSignalDataSaving(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	trial, task := createTestTrial(t, api, curUser)

	activeConfig, err := api.m.db.ActiveExperimentConfig(trial.ExperimentID)
	require.NoError(t, err)

	signal := "sub"
	activeConfig.RawLogPolicies = &expconf.LogPoliciesConfig{
		expconf.LogPolicy{RawPattern: "sub", RawSignal: &signal},
		expconf.LogPolicy{RawPattern: `\d{5}$`},
	}

	v, err := json.Marshal(activeConfig)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(v, &m))

	_, err = db.Bun().NewUpdate().Table("experiments").
		Where("id = ?", trial.ExperimentID).
		Set("config = ?", m).
		Exec(ctx)
	require.NoError(t, err)

	_, err = api.PostTaskLogs(ctx, &apiv1.PostTaskLogsRequest{
		Logs: []*taskv1.TaskLog{
			{
				TaskId:  string(task.TaskID),
				AgentId: ptrs.Ptr("a1"),
				Log:     "stringsubstring",
			},
			{
				TaskId:  string(task.TaskID),
				AgentId: ptrs.Ptr("a1"),
				Log:     "12345",
			},
		},
	})
	require.NoError(t, err)

	runsOut := struct {
		bun.BaseModel `bun:"table:runs"`
		LogSignal     *string `db:"log_signal"`
	}{}

	err = db.Bun().NewSelect().Model(&runsOut).
		Where("id = ?", trial.ID).
		Scan(ctx)
	require.NoError(t, err)
	require.NotNil(t, runsOut)
	require.NotNil(t, runsOut.LogSignal)

	require.Equal(t, "sub", *runsOut.LogSignal)

	tasksOut := struct {
		bun.BaseModel `bun:"table:tasks"`
		LogSignal     *string `db:"log_signal"`
	}{}
	err = db.Bun().NewSelect().Model(&tasksOut).
		Join("LEFT JOIN run_id_task_id AS rt on tasks.task_id = rt.task_id").
		Where("run_id = ?", trial.ID).
		Scan(ctx)
	require.NoError(t, err)
	require.NotNil(t, tasksOut)
	require.NotNil(t, tasksOut.LogSignal)

	require.Equal(t, "sub", *tasksOut.LogSignal)
}

//go:build integration
// +build integration

package internal

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/logpattern"
	"github.com/determined-ai/determined/master/internal/sproto"
	taskPkg "github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
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
}

func validateResponseSummary(t *testing.T, expectedTaskSummary *taskv1.AllocationSummary,
	taskSummary *taskv1.AllocationSummary,
) {
	require.Equal(t, expectedTaskSummary.TaskId, taskSummary.TaskId)
	require.Equal(t, expectedTaskSummary.AllocationId, taskSummary.AllocationId)
	require.Equal(t, expectedTaskSummary.Name, taskSummary.Name)

	require.Equal(t, len(expectedTaskSummary.Resources), len(taskSummary.Resources))

	for i, r := range taskSummary.Resources {
		expectedResource := expectedTaskSummary.Resources[i]
		require.Equal(t, expectedResource.ResourcesId, r.ResourcesId)
		require.Equal(t, expectedResource.AllocationId, r.AllocationId)
		var contID string = *r.ContainerId
		var expectedContID string = *expectedResource.ContainerId
		require.Equal(t, expectedContID, contID)
		require.Equal(t, len(expectedResource.AgentDevices), len(r.AgentDevices))
		require.ElementsMatch(t, maps.Keys(expectedResource.AgentDevices),
			maps.Keys(r.AgentDevices))
		for devName, devs := range expectedResource.AgentDevices {
			for ind, dev := range devs.Devices {
				agentDevice := r.AgentDevices[devName].Devices[ind]
				require.Equal(t, dev.Brand, agentDevice.Brand)
				require.Equal(t, dev.Uuid, agentDevice.Uuid)
			}
		}
	}

	require.Equal(t, len(expectedTaskSummary.ProxyPorts), len(taskSummary.ProxyPorts))
	for ind, pp := range taskSummary.ProxyPorts {
		expProxyPortConfig := expectedTaskSummary.ProxyPorts[ind]
		require.Equal(t, expProxyPortConfig.ServiceId, pp.ServiceId)
		require.Equal(t, int(expProxyPortConfig.Port), int(pp.Port))
		require.Equal(t, expProxyPortConfig.ProxyTcp, pp.ProxyTcp)
		require.Equal(t, expProxyPortConfig.Unauthenticated, pp.Unauthenticated)
	}
}

func CheckObfuscatedTask(t *testing.T, taskSummary *taskv1.AllocationSummary, id string,
	allocs map[model.AllocationID]sproto.AllocationSummary, permissions []string,
) error {
	hasPermissions := slices.Contains(permissions, id)
	allocSummary := allocs[model.AllocationID(id)]
	if hasPermissions {
		validateResponseSummary(t, allocSummary.Proto(), taskSummary)
	} else {
		// Create expected obfuscated task summary.
		obfuscatedSummary := &taskv1.AllocationSummary{}
		obfuscatedSummary.TaskId = authz2.HiddenString
		obfuscatedSummary.AllocationId = authz2.HiddenString
		obfuscatedSummary.Name = ""

		for _, r := range taskSummary.Resources {
			resource := &taskv1.ResourcesSummary{}
			resource.ResourcesId = authz2.HiddenString
			resource.AllocationId = authz2.HiddenString
			var contID string = string(authz2.HiddenString)
			resource.ContainerId = &contID
			agentDevice := make(map[string]*taskv1.ResourcesSummary_Devices)
			for _, devs := range r.AgentDevices {
				obfuscatedDevs := &taskv1.ResourcesSummary_Devices{}
				obfuscatedDevices := make([]*devicev1.Device, len(devs.Devices))
				for ind := range devs.Devices {
					obfuscatedDev := &devicev1.Device{
						Brand: authz2.HiddenString,
						Uuid:  authz2.HiddenString,
					}
					obfuscatedDevices[ind] = obfuscatedDev
				}
				obfuscatedDevs.Devices = obfuscatedDevices
				agentDevice[authz2.HiddenString] = obfuscatedDevs
			}
			resource.AgentDevices = agentDevice
			obfuscatedSummary.Resources = append(obfuscatedSummary.Resources, resource)
		}
		proxyPortConfs := make([]*taskv1.ProxyPortConfig, len(taskSummary.ProxyPorts))
		for ind := range taskSummary.ProxyPorts {
			ppConf := &taskv1.ProxyPortConfig{}
			ppConf.ServiceId = authz2.HiddenString
			ppConf.Port = authz2.HiddenInt
			ppConf.ProxyTcp = authz2.HiddenBool
			ppConf.Unauthenticated = authz2.HiddenBool
			proxyPortConfs[ind] = ppConf
		}
		obfuscatedSummary.ProxyPorts = proxyPortConfs
		validateResponseSummary(t, obfuscatedSummary, taskSummary)
	}
	return nil
}

func mockNotebookWithWorkspaceID(
	ctx context.Context, api *apiServer, t *testing.T, workspaceID int,
) (model.TaskID, model.AllocationID) {
	nb := &model.Task{
		TaskID:   model.NewTaskID(),
		TaskType: model.TaskTypeNotebook,
	}
	require.NoError(t, api.m.db.AddTask(nb))

	allocationID := model.AllocationID(string(nb.TaskID) + ".1")
	require.NoError(t, api.m.db.AddAllocation(&model.Allocation{
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

	return nb.TaskID, allocationID
}

func getRandomString() string {
	randomUUID := uuid.New()
	return randomUUID.String()
}

func mockAllocationSummary(t *testing.T, taskID model.TaskID,
	allocID model.AllocationID,
) sproto.AllocationSummary {
	grs := getRandomString
	summary := sproto.AllocationSummary{
		TaskID: taskID, AllocationID: allocID, Name: grs(),
	}

	// Since mockAllocationSummary is used in conjunction with testing obfuscated tasks and
	// obfuscating a boolean just sets it to false, we set defaultBool to true so that we can
	// test whether or not the value gets obscured.
	defaultBool := true
	pPortConf := sproto.ProxyPortConfig{
		ServiceID: grs(),
		Port:      0, ProxyTCP: defaultBool, Unauthenticated: defaultBool,
	}
	pPortConfs := []*sproto.ProxyPortConfig{&pPortConf}

	resID := sproto.ResourcesID(grs())
	restype := sproto.ResourcesType(grs())
	agentDevices := make(map[aproto.ID][]device.Device)
	aPID := aproto.ID(grs())
	var devs []device.Device

	// Same as above; since obfuscating an int just sets it to -1, we set devID to any
	// non-negative int so that we can test whether or not the value gets obscured.
	randID, err := rand.Int(rand.Reader, big.NewInt(100000000))
	require.NoError(t, err)
	devID := randID.Int64()
	dev := device.Device{
		ID: device.ID(devID), Brand: grs(), UUID: grs(), Type: device.Type(grs()),
	}
	devs = append(devs, dev)
	agentDevices[aPID] = devs
	resContID := cproto.ID(grs())
	resourcesSummary := sproto.ResourcesSummary{
		ResourcesID: resID, ResourcesType: restype, AllocationID: allocID,
		AgentDevices: agentDevices, ContainerID: &resContID,
	}
	resourcesSummaries := []sproto.ResourcesSummary{resourcesSummary}

	summary.Resources = resourcesSummaries
	summary.ProxyPorts = pPortConfs

	return summary
}

func TestGetTasksAuthZ(t *testing.T) {
	var allocations map[model.AllocationID]sproto.AllocationSummary
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil, func(context *actor.Context) error {
		switch context.Message().(type) {
		case sproto.GetAllocationSummaries:
			context.Respond(allocations)
		}
		return nil
	})
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

	canAccessNotebookID, canAccessAllocationID := mockNotebookWithWorkspaceID(ctx, api, t, -100)
	authZNSC.On("CanGetNSC", mock.Anything, mock.Anything, model.AccessScopeID(-100)).
		Return(nil).Once()

	cantAccessNotebookID, cantAccessAllocationID := mockNotebookWithWorkspaceID(ctx, api, t, -101)
	authZNSC.On("CanGetNSC", mock.Anything, mock.Anything, model.AccessScopeID(-101)).
		Return(authz2.PermissionDeniedError{}).Once()

	cantAccessNotebookID2, cantAccessAllocationID2 := mockNotebookWithWorkspaceID(ctx, api, t, -102)
	authZNSC.On("CanGetNSC", mock.Anything, mock.Anything, model.AccessScopeID(-102)).
		Return(authz2.PermissionDeniedError{}).Once()

	summaryAccess := mockAllocationSummary(t, canAccessNotebookID, canAccessAllocationID)
	summaryNoAccess := mockAllocationSummary(t, cantAccessNotebookID, cantAccessAllocationID)
	summaryNoAccess2 := mockAllocationSummary(t, cantAccessNotebookID2, cantAccessAllocationID2)

	allocations = map[model.AllocationID]sproto.AllocationSummary{
		"alloc0": {
			TaskID: expCanAccessTask.TaskID,
		},
		"alloc1": {
			TaskID: expCantAccessTask.TaskID,
		},
		"alloc2": summaryAccess,
		"alloc3": summaryNoAccess,
		"alloc4": summaryNoAccess2,
	}

	NTSCTasks := make(map[string]int)
	NTSCTasks["alloc2"] = 0
	NTSCTasks[authz2.HiddenString] = 0

	resp, err := api.GetTasks(ctx, &apiv1.GetTasksRequest{})
	require.NoError(t, err)

	// require.ElementsMatch(t, []string{"alloc0", "alloc1", "alloc2", authz2.HiddenString},
	// 	maps.Keys(resp.AllocationIdToSummary))
	require.Equal(t, len(allocations), len(resp.AllocationIdToSummary))

	savedAllocIDs := map[string]bool{"alloc0": true, "alloc1": true, "alloc2": true,
		"alloc3": false, "alloc4": false}
	for id, shouldContainID := range savedAllocIDs {
		_, containsID := resp.AllocationIdToSummary[id]
		require.Equal(t, shouldContainID, containsID)
	}

	// Check that NTSC tasks that are accessed by user with no permissions are obfuscated.
	permissions := []string{"alloc2"}
	for id, summary := range resp.AllocationIdToSummary {
		_, contains := NTSCTasks[id]
		if contains {
			err := CheckObfuscatedTask(t, summary, id, allocations, permissions)
			require.NoError(t, err)
		}
	}
}

func TestPostTaskLogsLogPattern(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	trial, task := createTestTrial(t, api, curUser)

	activeConfig, err := api.m.db.ActiveExperimentConfig(trial.ExperimentID)
	require.NoError(t, err)
	activeConfig.RawLogPolicies = expconf.LogPoliciesConfig{
		expconf.LogPolicy{RawPattern: "sub", RawAction: expconf.LogAction{
			RawCancelRetries: &expconf.LogActionCancelRetries{},
		}},
		expconf.LogPolicy{RawPattern: `\d{5}$`, RawAction: expconf.LogAction{
			RawExcludeNode: &expconf.LogActionExcludeNode{},
		}},
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
	require.NoError(t, api.m.db.AddTask(task), "failed to add task")

	aID := tID + "-1"
	a := &model.Allocation{
		TaskID:       tID,
		AllocationID: model.AllocationID(aID),
		Slots:        1,
		ResourcePool: "default",
	}
	require.NoError(t, api.m.db.AddAllocation(a), "failed to add allocation")
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
	require.Equal(t, len(resp.AcceleratorData), 1, "incorrect number of allocation accelerator data returned")
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
	require.NoError(t, api.m.db.AddTask(task), "failed to add task")

	aID := tID + "-1"
	a := &model.Allocation{
		TaskID:       tID,
		AllocationID: model.AllocationID(aID),
		Slots:        1,
		ResourcePool: "default",
	}
	require.NoError(t, api.m.db.AddAllocation(a), "failed to add allocation")

	resp, err := api.GetTaskAcceleratorData(ctx,
		&apiv1.GetTaskAcceleratorDataRequest{TaskId: tID.String()})
	require.NoError(t, err, "failed to get task AccelerationData")
	require.Equal(t, len(resp.AcceleratorData), 0, "unexpected allocation accelerator data returned")
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
	require.NoError(t, api.m.db.AddTask(task), "failed to add task")

	aID1 := tID + "-1"
	a1 := &model.Allocation{
		TaskID:       tID,
		AllocationID: model.AllocationID(aID1),
		Slots:        1,
		ResourcePool: "default",
	}
	require.NoError(t, api.m.db.AddAllocation(a1), "failed to add allocation")
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
	require.NoError(t, api.m.db.AddAllocation(a2), "failed to add allocation")

	resp, err := api.GetTaskAcceleratorData(ctx,
		&apiv1.GetTaskAcceleratorDataRequest{TaskId: tID.String()})
	require.NoError(t, err, "failed to get task AccelerationData")
	require.Equal(t, len(resp.AcceleratorData), 1, "incorrect number of allocation accelerator data returned")
	require.Equal(t, resp.AcceleratorData[0].AllocationId,
		aID1.String(), "failed to get the correct allocation's accelerator data")
}

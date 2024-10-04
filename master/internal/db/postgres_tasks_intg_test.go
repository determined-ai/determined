//go:build integration
// +build integration

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

// TestJobTaskAndAllocationAPI, in lieu of an ORM, ensures that the mappings into and out of the
// database are total. We should look into an ORM in the near to medium term future.
func TestJobTaskAndAllocationAPI(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	// Add a mock user.
	user := RequireMockUser(t, db)

	// Add a job.
	jID := model.NewJobID()
	jIn := &model.Job{
		JobID:   jID,
		JobType: model.JobTypeExperiment,
		OwnerID: &user.ID,
		QPos:    decimal.New(0, 0),
	}
	err := AddJob(jIn)
	require.NoError(t, err, "failed to add job")

	// Retrieve it back and make sure the mapping is exhaustive.
	jOut, err := JobByID(context.TODO(), jID)
	require.NoError(t, err, "failed to retrieve job")
	require.True(t, reflect.DeepEqual(jIn, jOut), pprintedExpect(jIn, jOut))

	// Add a task.
	tID := model.NewTaskID()
	tIn := &model.Task{
		TaskID:    tID,
		JobID:     &jID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}
	err = AddTask(ctx, tIn)
	require.NoError(t, err, "failed to add task")

	// Retrieve it back and make sure the mapping is exhaustive.
	tOut, err := TaskByID(ctx, tID)
	require.NoError(t, err, "failed to retrieve task")
	require.True(t, reflect.DeepEqual(tIn, tOut), pprintedExpect(tIn, tOut))

	// Complete it.
	tIn.EndTime = ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond))
	err = CompleteTask(ctx, tID, *tIn.EndTime)
	require.NoError(t, err, "failed to mark task completed")

	// Re-retrieve it back and make sure the mapping is still exhaustive.
	tOut, err = TaskByID(ctx, tID)
	require.NoError(t, err, "failed to re-retrieve task")
	require.True(t, reflect.DeepEqual(tIn, tOut), pprintedExpect(tIn, tOut))

	// And an allocation.
	ports := map[string]int{}
	ports["dtrain_port"] = 0
	ports["inter_train_process_comm_port1"] = 0
	ports["inter_train_process_comm_port2"] = 0
	ports["c10d_port"] = 0

	aID := model.AllocationID(string(tID) + "-1")
	aIn := &model.Allocation{
		AllocationID: aID,
		TaskID:       tID,
		Slots:        8,
		ResourcePool: "somethingelse",
		StartTime:    ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond)),
		Ports:        ports,
	}
	err = AddAllocation(ctx, aIn)
	require.NoError(t, err, "failed to add allocation")

	// Update ports
	ports["dtrain_port"] = 0
	ports["inter_train_process_comm_port1"] = 0
	ports["inter_train_process_comm_port2"] = 0
	ports["c10d_port"] = 0
	aIn.Ports = ports
	err = UpdateAllocationPorts(ctx, *aIn)
	require.NoError(t, err, "failed to update port offset")

	// Retrieve it back and make sure the mapping is exhaustive.
	aOut, err := AllocationByID(ctx, aIn.AllocationID)
	require.NoError(t, err, "failed to retrieve allocation")
	require.True(t, reflect.DeepEqual(aIn, aOut), pprintedExpect(aIn, aOut))

	// Complete it.
	aIn.EndTime = ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond))
	err = CompleteAllocation(ctx, aIn)
	require.NoError(t, err, "failed to mark allocation completed")

	// Re-retrieve it back and make sure the mapping is still exhaustive.
	aOut, err = AllocationByID(ctx, aIn.AllocationID)
	require.NoError(t, err, "failed to re-retrieve allocation")
	require.True(t, reflect.DeepEqual(aIn, aOut), pprintedExpect(aIn, aOut))
}

func TestRecordAndEndTaskStats(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	tID := model.NewTaskID()
	require.NoError(t, AddTask(ctx, &model.Task{
		TaskID:    tID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}), "failed to add task")

	allocationID := model.AllocationID(tID + "allocationID")
	require.NoError(t, AddAllocation(ctx, &model.Allocation{
		TaskID:       tID,
		AllocationID: allocationID,
	}), "failed to add allocation")

	var expected []*model.TaskStats
	for i := 0; i < 3; i++ {
		taskStats := &model.TaskStats{
			AllocationID: allocationID,
			EventType:    "IMAGEPULL",
			ContainerID:  ptrs.Ptr(cproto.NewID()),
			StartTime:    ptrs.Ptr(time.Now().Truncate(time.Millisecond)),
		}
		if i == 0 {
			taskStats.ContainerID = nil
		}
		require.NoError(t, RecordTaskStats(ctx, taskStats))

		taskStats.EndTime = ptrs.Ptr(time.Now().Truncate(time.Millisecond))
		require.NoError(t, RecordTaskEndStats(ctx, taskStats))
		expected = append(expected, taskStats)
	}

	var actual []*model.TaskStats
	err := Bun().NewSelect().
		Model(&actual).
		Where("allocation_id = ?", allocationID).
		Scan(context.TODO(), &actual)
	require.NoError(t, err)

	require.ElementsMatch(t, expected, actual)

	err = EndAllTaskStats(ctx)
	require.NoError(t, err)
}

func TestNonExperimentTasksContextDirectory(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	// Task doesn't exist.
	_, err := NonExperimentTasksContextDirectory(ctx, model.TaskID(uuid.New().String()))
	require.ErrorIs(t, err, sql.ErrNoRows)

	// Nil context directory.
	tID := model.NewTaskID()
	require.NoError(t, AddTask(ctx, &model.Task{
		TaskID:    tID,
		TaskType:  model.TaskTypeNotebook,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}), "failed to add task")

	require.NoError(t, AddNonExperimentTasksContextDirectory(ctx, tID, nil))

	dir, err := NonExperimentTasksContextDirectory(ctx, tID)
	require.NoError(t, err)
	require.Empty(t, dir)

	// Non nil context directory.
	tID = model.NewTaskID()
	require.NoError(t, AddTask(ctx, &model.Task{
		TaskID:    tID,
		TaskType:  model.TaskTypeNotebook,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}), "failed to add task")

	expectedDir := []byte{3, 2, 1}
	require.NoError(t, AddNonExperimentTasksContextDirectory(ctx, tID, expectedDir))

	dir, err = NonExperimentTasksContextDirectory(ctx, tID)
	require.NoError(t, err)
	require.Equal(t, expectedDir, dir)
}

func TestAllocationState(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	// Add an allocation of every possible state.
	states := []model.AllocationState{
		model.AllocationStatePending,
		model.AllocationStateAssigned,
		model.AllocationStatePulling,
		model.AllocationStateStarting,
		model.AllocationStateRunning,
		model.AllocationStateTerminating,
		model.AllocationStateTerminated,
	}
	for _, state := range states {
		tID := model.NewTaskID()
		task := &model.Task{
			TaskID:    tID,
			TaskType:  model.TaskTypeTrial,
			StartTime: time.Now().UTC().Truncate(time.Millisecond),
		}
		require.NoError(t, AddTask(ctx, task), "failed to add task")

		a := &model.Allocation{
			TaskID:       tID,
			AllocationID: model.AllocationID(tID + "allocationID"),
			ResourcePool: "default",
			State:        &state,
		}
		require.NoError(t, AddAllocation(ctx, a), "failed to add allocation")

		// Update allocation to every possible state.
		testNoUpdate := true
		for j := 0; j < len(states); j++ {
			if testNoUpdate {
				testNoUpdate = false
				j-- // Go to first iteration of loop after this.
			} else {
				a.State = &states[j]
				require.NoError(t, UpdateAllocationState(ctx, *a),
					"failed to update allocation state")
			}

			// Get task back as a proto struct.
			tOut := &taskv1.Task{}
			require.NoError(t, db.QueryProto("get_task", tOut, tID), "failed to get task")

			// Ensure our state is the same as allocation.
			require.Len(t, tOut.Allocations, 1, "failed to get exactly 1 allocation")
			aOut := tOut.Allocations[0]

			if slices.Contains([]model.AllocationState{
				model.AllocationStatePending,
				model.AllocationStateAssigned,
			}, *a.State) {
				require.Equal(t, "STATE_QUEUED", aOut.State.String(),
					"allocation states not converted to queued")
			} else {
				require.Equal(t, a.State.Proto(), aOut.State, "proto state not equal")
				require.Equal(t, fmt.Sprintf("STATE_%s", *a.State), aOut.State.String(),
					"proto state to strings not equal")
			}
		}
	}
}

func TestExhaustiveEnums(t *testing.T) {
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	type check struct {
		goType          string
		goMembers       map[string]bool
		postgresType    string
		postgresMembers map[string]bool
		ignore          map[string]bool
	}
	db := SingleDB()

	checks := map[string]*check{}
	addCheck := func(goType, postgresType string, ignore map[string]bool) {
		checks[goType] = &check{
			goType:          goType,
			goMembers:       map[string]bool{},
			postgresType:    postgresType,
			postgresMembers: map[string]bool{},
			ignore:          ignore,
		}
	}
	addCheck("JobType", "public.job_type", map[string]bool{})
	addCheck("TaskType", "public.task_type", map[string]bool{})
	addCheck("State", "public.experiment_state", map[string]bool{
		"PARTIALLY_DELETED": true,
		"DELETED":           true,
	})
	addCheck("AllocationState", "public.allocation_state", map[string]bool{})

	// Populate postgres types.
	for _, c := range checks {
		q := fmt.Sprintf("SELECT unnest(enum_range(NULL::%s))::text", c.postgresType)
		rows, err := db.sql.Queryx(q)
		require.NoError(t, err, "querying postgres enum members")
		defer rows.Close()
		for rows.Next() {
			var text string
			require.NoError(t, rows.Scan(&text), "scanning enum value")
			c.postgresMembers[text] = true
		}
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "../../pkg/model", nil, parser.ParseComments)
	require.NoError(t, err)
	for _, p := range pkgs {
		for _, f := range p.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				vs, ok := n.(*ast.ValueSpec)
				if !ok {
					return true
				}

				vsTypeIdent, ok := vs.Type.(*ast.Ident)
				if !ok {
					return true
				}

				c, ok := checks[vsTypeIdent.Name]
				if !ok {
					return true
				}

				// We can error out now because we're certainly on something we want to check.
				for _, v := range vs.Values {
					bl, ok := v.(*ast.BasicLit)
					require.True(t, ok, "linter can only handle pg enums as basic lits")
					require.Equal(t, token.STRING, bl.Kind, "linter can only handle lit strings")
					c.goMembers[strings.Trim(bl.Value, "\"'`")] = true
				}

				return true
			})
		}
	}

	for _, c := range checks {
		for name := range c.ignore {
			delete(c.postgresMembers, name)
			delete(c.goMembers, name)
		}

		pb, err := json.Marshal(c.postgresMembers)
		require.NoError(t, err)
		gb, err := json.Marshal(c.goMembers)
		require.NoError(t, err)

		// Gives pretty diff.
		require.JSONEq(t, string(pb), string(gb))
	}
}

func TestAddTask(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	u := RequireMockUser(t, db)
	jID := RequireMockJob(t, db, &u.ID)

	// Add a task.
	tID := model.NewTaskID()
	tIn := &model.Task{
		TaskID:     tID,
		JobID:      &jID,
		TaskType:   model.TaskTypeTrial,
		StartTime:  time.Now().UTC().Truncate(time.Millisecond),
		LogVersion: model.TaskLogVersion0,
	}
	err := AddTask(ctx, tIn)
	require.NoError(t, err, "failed to add task")

	// Check that task is added to the db & test TaskByID.
	task, err := TaskByID(ctx, tIn.TaskID)
	require.NoError(t, err)
	require.Equal(t, tIn, task)
}

func TestTaskCompleted(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	tIn := RequireMockTask(t, db, nil)

	completed, err := TaskCompleted(ctx, tIn.TaskID)
	require.False(t, completed)
	require.NoError(t, err)

	err = CompleteTask(ctx, tIn.TaskID, time.Now().UTC().Truncate(time.Millisecond))
	require.NoError(t, err)

	completed, err = TaskCompleted(ctx, tIn.TaskID)
	require.True(t, completed)
	require.NoError(t, err)
}

func TestAddAllocation(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	tIn := RequireMockTask(t, db, nil)
	a := model.Allocation{
		AllocationID: model.AllocationID(fmt.Sprintf("%s-1", tIn.TaskID)),
		TaskID:       tIn.TaskID,
		StartTime:    ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond)),
		State:        ptrs.Ptr(model.AllocationStateTerminated),
	}

	err := AddAllocation(ctx, &a)
	require.NoError(t, err, "failed to add allocation")

	res, err := AllocationByID(ctx, a.AllocationID)
	require.NoError(t, err)
	require.Equal(t, a.AllocationID, res.AllocationID)
	require.Equal(t, a.TaskID, res.TaskID)
	require.Equal(t, a.StartTime, res.StartTime)
	require.Equal(t, a.State, res.State)
}

func TestAddAllocationExitStatus(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	tIn := RequireMockTask(t, db, nil)
	aIn := RequireMockAllocation(t, db, tIn.TaskID)

	statusCode := int32(1)
	exitReason := "testing-exit-reason"
	exitErr := "testing-exit-err"

	aIn.ExitReason = &exitReason
	aIn.ExitErr = &exitErr
	aIn.StatusCode = &statusCode

	err := AddAllocationExitStatus(ctx, aIn)
	require.NoError(t, err)

	res, err := AllocationByID(ctx, aIn.AllocationID)
	require.NoError(t, err)
	require.Equal(t, aIn.ExitErr, res.ExitErr)
	require.Equal(t, aIn.ExitReason, res.ExitReason)
	require.Equal(t, aIn.StatusCode, res.StatusCode)
}

func TestCompleteAllocation(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	tIn := RequireMockTask(t, db, nil)
	aIn := RequireMockAllocation(t, db, tIn.TaskID)

	aIn.EndTime = ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond))

	err := CompleteAllocation(ctx, aIn)
	require.NoError(t, err)

	res, err := AllocationByID(ctx, aIn.AllocationID)
	require.NoError(t, err)
	require.Equal(t, aIn.EndTime, res.EndTime)
}

func TestCompleteAllocationTelemetry(t *testing.T) {
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	tIn := RequireMockTask(t, db, nil)
	aIn := RequireMockAllocation(t, db, tIn.TaskID)

	bytes, err := CompleteAllocationTelemetry(context.TODO(), aIn.AllocationID)
	require.NoError(t, err)
	require.Contains(t, string(bytes), string(aIn.AllocationID))
	require.Contains(t, string(bytes), string(*tIn.JobID))
	require.Contains(t, string(bytes), string(tIn.TaskType))
}

func TestAllocationByID(t *testing.T) {
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	tIn := RequireMockTask(t, db, nil)
	aIn := RequireMockAllocation(t, db, tIn.TaskID)

	a, err := AllocationByID(context.TODO(), aIn.AllocationID)
	require.NoError(t, err)
	require.Equal(t, aIn, a)
}

func TestAllocationSessionFlow(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	uIn := RequireMockUser(t, db)
	tIn := RequireMockTask(t, db, nil)
	aIn := RequireMockAllocation(t, db, tIn.TaskID)

	tok, err := StartAllocationSession(ctx, aIn.AllocationID, &uIn)
	require.NoError(t, err)
	require.NotNil(t, tok)

	as, err := allocationSessionByID(aIn.AllocationID)
	require.NoError(t, err)
	require.Equal(t, uIn.ID, *as.OwnerID)

	running := model.AllocationStatePulling
	aIn.State = &running
	err = UpdateAllocationState(ctx, *aIn)
	require.NoError(t, err)

	a, err := AllocationByID(ctx, aIn.AllocationID)
	require.NoError(t, err)
	require.Equal(t, aIn.State, a.State)

	err = DeleteAllocationSession(ctx, aIn.AllocationID)
	require.NoError(t, err)

	as, err = allocationSessionByID(aIn.AllocationID)
	require.ErrorContains(t, err, "no rows in result set")
	require.Nil(t, as)
}

func TestUpdateAllocation(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	tIn := RequireMockTask(t, db, nil)
	aIn := RequireMockAllocation(t, db, tIn.TaskID)

	// Testing UpdateAllocation Ports
	aIn.Ports = map[string]int{"abc": 123, "def": 456}
	err := UpdateAllocationPorts(ctx, *aIn)
	require.NoError(t, err)

	a, err := AllocationByID(ctx, aIn.AllocationID)
	require.NoError(t, err)
	require.Equal(t, aIn.Ports, a.Ports)

	// Testing UpdateAllocationStartTime
	newStartTime := ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond))
	aIn.StartTime = newStartTime

	err = UpdateAllocationStartTime(ctx, *aIn)
	require.NoError(t, err)

	a, err = AllocationByID(ctx, aIn.AllocationID)
	require.NoError(t, err)
	require.Equal(t, aIn.StartTime, a.StartTime)

	// Testing UpdateAllocationProxyAddress
	proxyAddr := "here"
	aIn.ProxyAddress = &proxyAddr

	err = UpdateAllocationProxyAddress(ctx, *aIn)
	require.NoError(t, err)

	a, err = AllocationByID(ctx, aIn.AllocationID)
	require.NoError(t, err)
	require.Equal(t, aIn.ProxyAddress, a.ProxyAddress)
}

func TestCloseOpenAllocations(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	// Create test allocations, with a NULL end time.
	t1In := RequireMockTask(t, db, nil)
	a1In := RequireMockAllocation(t, db, t1In.TaskID)

	t2In := RequireMockTask(t, db, nil)
	a2In := RequireMockAllocation(t, db, t2In.TaskID)

	// Set status for both open allocation as 'terminated'.
	terminated := model.AllocationStateTerminated
	a1In.State = &terminated
	a2In.State = &terminated

	// Close only a2In open allocations (filter out the rest).
	err := CloseOpenAllocations(ctx, []model.AllocationID{a1In.AllocationID})
	require.NoError(t, err)

	a1, err := AllocationByID(ctx, a1In.AllocationID)
	require.NoError(t, err)
	require.Nil(t, a1.EndTime)

	a2, err := AllocationByID(ctx, a2In.AllocationID)
	require.NoError(t, err)
	require.NotNil(t, a2.EndTime)

	// Close the rest of the open allocations.
	err = CloseOpenAllocations(ctx, []model.AllocationID{})
	require.NoError(t, err)

	a1, err = AllocationByID(ctx, a1In.AllocationID)
	require.NoError(t, err)
	require.NotNil(t, a1.EndTime)
}

func TestTaskLogsFlow(t *testing.T) {
	pgDB, closeDB := MustResolveTestPostgres(t)
	defer closeDB()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	db := SingleDB()

	t1In := RequireMockTask(t, db, nil)
	t2In := RequireMockTask(t, db, nil)

	// Test AddTaskLogs & TaskLogCounts
	taskLog1 := RequireMockTaskLog(t, db, t1In.TaskID, "1")
	taskLog2 := RequireMockTaskLog(t, db, t1In.TaskID, "2")
	taskLog3 := RequireMockTaskLog(t, db, t2In.TaskID, "3")

	// Try adding only taskLog1, and count only 1 log.
	err := db.AddTaskLogs([]*model.TaskLog{taskLog1})
	require.NoError(t, err)

	// Try filtering by agentID & taskID -- only 1 exists.
	count, err := db.TaskLogsCount(t1In.TaskID, []api.Filter{{
		Field:     "agent_id",
		Operation: api.FilterOperationIn,
		Values:    []string{"testing-agent-1"},
	}})
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Try filtering by agentID & taskID -- none exist with this combination.
	count, err = db.TaskLogsCount(t2In.TaskID, []api.Filter{{
		Field:     "agent_id",
		Operation: api.FilterOperationIn,
		Values:    []string{"testing-agent-1"},
	}})
	require.NoError(t, err)
	require.Zero(t, count)

	// Try adding the rest of the Task logs, and count 2 for t1In.TaskID, and 1 for t2In.TaskID
	err = db.AddTaskLogs([]*model.TaskLog{taskLog2, taskLog3})
	require.NoError(t, err)

	count, err = db.TaskLogsCount(t1In.TaskID, []api.Filter{})
	require.NoError(t, err)
	require.Equal(t, 2, count)

	count, err = db.TaskLogsCount(t2In.TaskID, []api.Filter{})
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Test TaskLogsFields.
	resp, err := db.TaskLogsFields(t1In.TaskID)
	require.NoError(t, err)
	require.ElementsMatch(t, resp.AgentIds, []string{"testing-agent-1", "testing-agent-2"})
	require.ElementsMatch(t, resp.ContainerIds, []string{"1", "2"})

	// Test TaskLogs.
	// Get 1 task log matching t1In task ID.
	logs, _, err := db.TaskLogs(t1In.TaskID, 1, []api.Filter{}, apiv1.OrderBy_ORDER_BY_UNSPECIFIED, nil)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, logs[0].TaskID, string(t1In.TaskID))
	require.Contains(t, []string{"1", "2"}, *logs[0].ContainerID)

	// Get up to 5 tasks matching t2In task ID -- receive only 2.
	logs, _, err = db.TaskLogs(t1In.TaskID, 5, []api.Filter{}, apiv1.OrderBy_ORDER_BY_UNSPECIFIED, nil)
	require.NoError(t, err)
	require.Len(t, logs, 2)

	// Filter by search text.
	logs, _, err = db.TaskLogs(t1In.TaskID, 5, []api.Filter{{
		Field:     "log",
		Operation: api.FilterOperationStringContainment,
		Values:    []string{"this"},
	}}, apiv1.OrderBy_ORDER_BY_UNSPECIFIED, nil)
	require.NoError(t, err)
	require.Len(t, logs, 2)

	logs, _, err = db.TaskLogs(t1In.TaskID, 5, []api.Filter{{
		Field:     "log",
		Operation: api.FilterOperationStringContainment,
		Values:    []string{"^th.s"},
	}}, apiv1.OrderBy_ORDER_BY_UNSPECIFIED, nil)
	require.NoError(t, err)
	require.Empty(t, logs)

	logs, _, err = db.TaskLogs(t1In.TaskID, 5, []api.Filter{{
		Field:     "log",
		Operation: api.FilterOperationRegexContainment,
		Values:    []string{"^th.s"},
	}}, apiv1.OrderBy_ORDER_BY_UNSPECIFIED, nil)
	require.NoError(t, err)
	require.Len(t, logs, 2)

	// Test DeleteTaskLogs.
	err = db.DeleteTaskLogs([]model.TaskID{t2In.TaskID})
	require.NoError(t, err)

	count, err = db.TaskLogsCount(t2In.TaskID, []api.Filter{})
	require.NoError(t, err)
	require.Zero(t, count)
}

func RequireMockTaskLog(t *testing.T, db *PgDB, tID model.TaskID, suffix string) *model.TaskLog {
	mockA := RequireMockAllocation(t, db, tID)
	agentID := "testing-agent-" + suffix
	containerID := suffix
	log := &model.TaskLog{
		TaskID:       string(tID),
		AllocationID: (*string)(&mockA.AllocationID),
		Log:          fmt.Sprintf("this is a log for task %s-%s", tID, suffix),
		AgentID:      &agentID,
		ContainerID:  &containerID,
	}
	return log
}

func allocationSessionByID(aID model.AllocationID) (*model.AllocationSession, error) {
	var res model.AllocationSession
	if err := Bun().NewSelect().Table("allocation_sessions").
		Where("allocation_id = ?", aID).Scan(context.TODO(), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

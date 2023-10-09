//go:build integration
// +build integration

package db

import (
	"context"
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

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

// TestJobTaskAndAllocationAPI, in lieu of an ORM, ensures that the mappings into and out of the
// database are total. We should look into an ORM in the near to medium term future.
func TestJobTaskAndAllocationAPI(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

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
	err := db.AddJob(jIn)
	require.NoError(t, err, "failed to add job")

	// Retrieve it back and make sure the mapping is exhaustive.
	jOut, err := db.JobByID(jID)
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
	err = db.AddTask(tIn)
	require.NoError(t, err, "failed to add task")

	// Retrieve it back and make sure the mapping is exhaustive.
	tOut, err := db.TaskByID(tID)
	require.NoError(t, err, "failed to retrieve task")
	require.True(t, reflect.DeepEqual(tIn, tOut), pprintedExpect(tIn, tOut))

	// Complete it.
	tIn.EndTime = ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond))
	err = db.CompleteTask(tID, *tIn.EndTime)
	require.NoError(t, err, "failed to mark task completed")

	// Re-retrieve it back and make sure the mapping is still exhaustive.
	tOut, err = db.TaskByID(tID)
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
	err = db.AddAllocation(aIn)
	require.NoError(t, err, "failed to add allocation")

	// Update ports
	ports["dtrain_port"] = 0
	ports["inter_train_process_comm_port1"] = 0
	ports["inter_train_process_comm_port2"] = 0
	ports["c10d_port"] = 0
	aIn.Ports = ports
	err = UpdateAllocationPorts(*aIn)
	require.NoError(t, err, "failed to update port offset")

	// Retrieve it back and make sure the mapping is exhaustive.
	aOut, err := db.AllocationByID(aIn.AllocationID)
	require.NoError(t, err, "failed to retrieve allocation")
	require.True(t, reflect.DeepEqual(aIn, aOut), pprintedExpect(aIn, aOut))

	// Complete it.
	aIn.EndTime = ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond))
	err = db.CompleteAllocation(aIn)
	require.NoError(t, err, "failed to mark allocation completed")

	// Re-retrieve it back and make sure the mapping is still exhaustive.
	aOut, err = db.AllocationByID(aIn.AllocationID)
	require.NoError(t, err, "failed to re-retrieve allocation")
	require.True(t, reflect.DeepEqual(aIn, aOut), pprintedExpect(aIn, aOut))
}

func TestRecordAndEndTaskStats(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	tID := model.NewTaskID()
	require.NoError(t, db.AddTask(&model.Task{
		TaskID:    tID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}), "failed to add task")

	allocationID := model.AllocationID(tID + "allocationID")
	require.NoError(t, db.AddAllocation(&model.Allocation{
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
		require.NoError(t, RecordTaskStatsBun(taskStats))

		taskStats.EndTime = ptrs.Ptr(time.Now().Truncate(time.Millisecond))
		require.NoError(t, RecordTaskEndStatsBun(taskStats))
		expected = append(expected, taskStats)
	}

	var actual []*model.TaskStats
	err := Bun().NewSelect().
		Model(&actual).
		Where("allocation_id = ?", allocationID).
		Scan(context.TODO(), &actual)
	require.NoError(t, err)

	require.ElementsMatch(t, expected, actual)
}

func TestAllocationState(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

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
		require.NoError(t, db.AddTask(task), "failed to add task")

		s := state
		a := &model.Allocation{
			TaskID:       tID,
			AllocationID: model.AllocationID(tID + "allocationID"),
			ResourcePool: "default",
			State:        &s,
		}
		require.NoError(t, db.AddAllocation(a), "failed to add allocation")

		// Update allocation to every possible state.
		testNoUpdate := true
		for j := 0; j < len(states); j++ {
			if testNoUpdate {
				testNoUpdate = false
				j-- // Go to first iteration of loop after this.
			} else {
				a.State = &states[j]
				require.NoError(t, db.UpdateAllocationState(*a),
					"failed to update allocation state")
			}

			// Get task back as a proto struct.
			tOut := &taskv1.Task{}
			require.NoError(t, db.QueryProto("get_task", tOut, tID), "failed to get task")

			// Ensure our state is the same as allocation.
			require.Equal(t, len(tOut.Allocations), 1, "failed to get exactly 1 allocation")
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
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	type check struct {
		goType          string
		goMembers       map[string]bool
		postgresType    string
		postgresMembers map[string]bool
		ignore          map[string]bool
	}
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

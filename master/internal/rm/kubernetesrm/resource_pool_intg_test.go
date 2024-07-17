package kubernetesrm

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	defaultResourcePool = "default"
)

type testLaunchOpts struct {
	name         string
	image        string
	entrypoint   []string
	aug          model.AgentUserGroup
	extraEnvVars map[string]string
	slots        int
	wantFailure  *sproto.ResourcesFailedError
}

func TestJobWorkflows(t *testing.T) {
	testCases := []testLaunchOpts{
		{
			name:        "single successful pod",
			entrypoint:  []string{"/bin/bash", "-c", "exit 0"},
			slots:       1,
			wantFailure: nil,
		},
		{
			name:         "extra env vars",
			entrypoint:   []string{"/bin/bash", "-c", "exit $DET_EXTRA_VAR"},
			extraEnvVars: map[string]string{"DET_EXTRA_VAR": "15"},
			slots:        1,
			wantFailure: &sproto.ResourcesFailedError{
				FailureType: sproto.ResourcesFailed,
				ExitCode:    (*sproto.ExitCode)(ptrs.Ptr(15)),
			},
		},
		{
			name:       "missing container image",
			image:      "lieblos/notanimageipushed",
			entrypoint: []string{"/bin/bash", "-c", "exit 0"},
			slots:      1,
			wantFailure: &sproto.ResourcesFailedError{
				FailureType: sproto.ResourcesFailed,
				ErrMsg:      "unrecoverable image pull errors",
			},
		},
		{
			name:       "single unsuccessful pod",
			entrypoint: []string{"/bin/bash", "-c", "exit 1"},
			slots:      1,
			wantFailure: &sproto.ResourcesFailedError{
				FailureType: sproto.ResourcesFailed,
				ExitCode:    (*sproto.ExitCode)(ptrs.Ptr(1)),
			},
		},
		{
			name:        "multiple successful pods",
			entrypoint:  []string{"/bin/bash", "-c", "exit 0"},
			slots:       2,
			wantFailure: nil,
		},
		{
			name:       "invalid job submission",
			entrypoint: []string{"exit 0"},
			aug: model.AgentUserGroup{
				UID: -1,
				GID: -1,
			},
			wantFailure: &sproto.ResourcesFailedError{
				FailureType: sproto.TaskError,
				ErrMsg:      "job crashed",
			},
		},
		{
			name:       "non-root users",
			entrypoint: []string{"/bin/bash", "-c", "exit $(id -u)"},
			aug: model.AgentUserGroup{
				UID: 123,
				GID: 123,
			},
			wantFailure: &sproto.ResourcesFailedError{
				FailureType: sproto.ResourcesFailed,
				ExitCode:    (*sproto.ExitCode)(ptrs.Ptr(123)),
			},
		},
		{
			name:        "long job", // Long enough to see all transitions.
			entrypoint:  []string{"/bin/bash", "-c", "sleep 10"},
			wantFailure: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testLaunch(t, tc)
		})
	}
}

func testLaunch(
	t *testing.T,
	opts testLaunchOpts,
) {
	if opts.image == "" {
		opts.image = "ubuntu:latest"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	id := uuid.NewString()
	jobID, taskID, allocationID := model.JobID(id), model.TaskID(id), model.AllocationID(id)
	startTime := time.Now()

	err := tasklist.GroupPriorityChangeRegistry.Add(jobID, func(i int) error { return nil })
	require.NoError(t, err)

	sub := rmevents.Subscribe(allocationID)
	rp.AllocateRequest(sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       opts.slots,
		ResourcePool:      "default",
	})

	require.Empty(t, rp.GetAllocationSummary(allocationID).Resources)
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID).Resources, 1)

	allocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.NotNil(t, allocated)
	require.Len(t, allocated.Resources, 1)

	for _, res := range allocated.Resources {
		conf := expconf.ExperimentConfig{ //nolint:exhaustruct
			RawEnvironment: &expconf.EnvironmentConfigV0{ //nolint:exhaustruct
				RawImage: &expconf.EnvironmentImageMapV0{ //nolint:exhaustruct
					RawCPU: &opts.image,
				},
			},
		}
		conf = schemas.WithDefaults(conf)

		err := res.Start(nil, tasks.TaskSpec{
			Description:     fmt.Sprintf("test-job-%s", uuid.NewString()[:8]),
			Entrypoint:      opts.entrypoint,
			AgentUserGroup:  &opts.aug,
			Environment:     conf.Environment(),
			ResourcesConfig: conf.Resources(),
			DontShipLogs:    true,
			ExtraEnvVars:    opts.extraEnvVars,
		}, sproto.ResourcesRuntimeInfo{})
		defer res.Kill(nil)
		require.NoError(t, err)
	}

	// Be careful to allow missing state changes here since the jobs are very short. It's
	// all good as long as we don't go backwards and end terminated.
	var stop *sproto.ResourcesStopped
	for state := sproto.Assigned; state != sproto.Terminated; {
		change := poll[*sproto.ResourcesStateChanged](ctx, t, sub)
		require.True(t, state.BeforeOrEqual(change.ResourcesState))
		state = change.ResourcesState
		stop = change.ResourcesStopped
	}

	require.NotNil(t, stop)
	if opts.wantFailure == nil {
		require.Nil(t, stop.Failure)
		return
	}
	require.NotNil(t, stop.Failure)
	assert.Equal(t, opts.wantFailure.FailureType, stop.Failure.FailureType)
	if opts.wantFailure.ExitCode != nil {
		assert.NotNil(t, stop.Failure.ExitCode)
		if stop.Failure.ExitCode != nil {
			assert.Equal(t, *opts.wantFailure.ExitCode, *stop.Failure.ExitCode)
		}
	} else {
		assert.Nil(t, stop.Failure.ExitCode)
	}
	assert.Contains(t, stop.Failure.ErrMsg, opts.wantFailure.ErrMsg)
}

func TestPodLogStreamerReattach(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	user := db.RequireMockUser(t, db.SingleDB())
	task := db.RequireMockTask(t, db.SingleDB(), &user.ID)
	alloc := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)
	allocationID, taskID, jobID := alloc.AllocationID, task.TaskID, *task.JobID
	startTime := task.StartTime

	err := tasklist.GroupPriorityChangeRegistry.Add(jobID, func(i int) error { return nil })
	require.NoError(t, err)

	sub := rmevents.Subscribe(allocationID)
	allocateReq := sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       1,
		ResourcePool:      "default",
	}
	rp.AllocateRequest(allocateReq)

	require.Empty(t, rp.GetAllocationSummary(allocationID).Resources)
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID).Resources, 1)

	allocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.NotNil(t, allocated)
	require.Len(t, allocated.Resources, 1)

	secret := uuid.NewString()
	for _, res := range allocated.Resources {
		conf := expconf.ExperimentConfig{ //nolint:exhaustruct
			RawEnvironment: &expconf.EnvironmentConfigV0{ //nolint:exhaustruct
				RawImage: &expconf.EnvironmentImageMapV0{ //nolint:exhaustruct
					RawCPU: ptrs.Ptr("ubuntu:latest"),
				},
			},
		}
		conf = schemas.WithDefaults(conf)

		err := res.Start(nil, tasks.TaskSpec{
			Description:     fmt.Sprintf("test-job-%s", uuid.NewString()[:8]),
			Entrypoint:      []string{"/bin/bash", "-c", fmt.Sprintf("sleep 15 && echo %s", secret)},
			AgentUserGroup:  &model.AgentUserGroup{},
			Environment:     conf.Environment(),
			ResourcesConfig: conf.Resources(),
			DontShipLogs:    true,
		}, sproto.ResourcesRuntimeInfo{})
		defer res.Kill(nil)
		require.NoError(t, err)
	}

	change := poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Pulling, change.ResourcesState)

	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Starting, change.ResourcesState)

	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Running, change.ResourcesState)
	require.NotNil(t, change.ResourcesStarted)

	// Remake all components and "reattach" to this new resource pool. This saves
	// us from needing to made the k8s code do graceful shutdown, but we should
	// do it anyway someday.
	rp = newTestResourcePool(newTestJobsService(t))

	sub = rmevents.Subscribe(allocationID)
	allocateReq.Restore = true
	rp.AllocateRequest(allocateReq)

	require.Empty(t, rp.GetAllocationSummary(allocationID).Resources)
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID).Resources, 1)

	reallocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.True(t, reallocated.Recovered)
	require.Len(t, reallocated.Resources, 1)

	seen := 0 // HACK: Because we don't have graceful shutdown, we have two log streamers up and get two events.
	for {
		log := poll[*sproto.ContainerLog](ctx, t, sub)
		if strings.Contains(log.Message(), secret) {
			t.Logf("saw one log: %s", log.Message())
			seen++
		}
		if seen == 2 {
			break
		}
	}
}

func TestPodLogStreamer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	id := uuid.NewString()
	jobID, taskID, allocationID := model.JobID(id), model.TaskID(id), model.AllocationID(id)
	startTime := time.Now()

	err := tasklist.GroupPriorityChangeRegistry.Add(jobID, func(i int) error { return nil })
	require.NoError(t, err)

	sub := rmevents.Subscribe(allocationID)
	rp.AllocateRequest(sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       1,
		ResourcePool:      "default",
	})

	require.Empty(t, rp.GetAllocationSummary(allocationID).Resources)
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID).Resources, 1)

	allocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.NotNil(t, allocated)
	require.Len(t, allocated.Resources, 1)

	require.Len(t, allocated.Resources, 1)
	secret := uuid.NewString()
	for _, res := range allocated.Resources {
		conf := expconf.ExperimentConfig{ //nolint:exhaustruct
			RawEnvironment: &expconf.EnvironmentConfigV0{ //nolint:exhaustruct
				RawImage: &expconf.EnvironmentImageMapV0{ //nolint:exhaustruct
					RawCPU: ptrs.Ptr("ubuntu:latest"),
				},
			},
		}
		conf = schemas.WithDefaults(conf)

		err := res.Start(nil, tasks.TaskSpec{
			Description:     fmt.Sprintf("test-job-%s", uuid.NewString()[:8]),
			Entrypoint:      []string{"/bin/bash", "-c", fmt.Sprintf("sleep 10 && echo %s", secret)},
			AgentUserGroup:  &model.AgentUserGroup{},
			Environment:     conf.Environment(),
			ResourcesConfig: conf.Resources(),
			DontShipLogs:    true,
		}, sproto.ResourcesRuntimeInfo{})
		defer res.Kill(nil)
		require.NoError(t, err)
	}

	for {
		log := poll[*sproto.ContainerLog](ctx, t, sub)
		if strings.Contains(log.Message(), secret) {
			return
		}
	}
}

func TestKill(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	id := uuid.NewString()
	jobID, taskID, allocationID := model.JobID(id), model.TaskID(id), model.AllocationID(id)
	startTime := time.Now()

	err := tasklist.GroupPriorityChangeRegistry.Add(jobID, func(i int) error { return nil })
	require.NoError(t, err)

	sub := rmevents.Subscribe(allocationID)
	rp.AllocateRequest(sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       1,
		ResourcePool:      "default",
	})

	require.Empty(t, rp.GetAllocationSummary(allocationID).Resources)
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID).Resources, 1)

	allocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.NotNil(t, allocated)
	require.Len(t, allocated.Resources, 1)

	for _, res := range allocated.Resources {
		conf := expconf.ExperimentConfig{ //nolint:exhaustruct
			RawEnvironment: &expconf.EnvironmentConfigV0{ //nolint:exhaustruct
				RawImage: &expconf.EnvironmentImageMapV0{ //nolint:exhaustruct
					RawCPU: ptrs.Ptr("ubuntu:latest"),
				},
			},
		}
		conf = schemas.WithDefaults(conf)

		err := res.Start(nil, tasks.TaskSpec{
			Description:     fmt.Sprintf("test-job-%s", uuid.NewString()[:8]),
			Entrypoint:      []string{"sleep", "99999"},
			AgentUserGroup:  &model.AgentUserGroup{},
			Environment:     conf.Environment(),
			ResourcesConfig: conf.Resources(),
			DontShipLogs:    true,
		}, sproto.ResourcesRuntimeInfo{})
		defer res.Kill(nil)
		require.NoError(t, err)
	}

	change := poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Pulling, change.ResourcesState)

	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Starting, change.ResourcesState)

	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Running, change.ResourcesState)
	require.NotNil(t, change.ResourcesStarted)

	for _, res := range allocated.Resources {
		res.Kill(nil)
	}

	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Terminated, change.ResourcesState)
	require.NotNil(t, change.ResourcesStopped)
	require.NotNil(t, change.ResourcesStopped.Failure)
	require.Contains(t, change.ResourcesStopped.Failure.ErrMsg, "kill")
}

func TestExternalKillWhileQueuedFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	id := uuid.NewString()
	jobID, taskID, allocationID := model.JobID(id), model.TaskID(id), model.AllocationID(id)
	startTime := time.Now()

	err := tasklist.GroupPriorityChangeRegistry.Add(jobID, func(i int) error { return nil })
	require.NoError(t, err)

	sub := rmevents.Subscribe(allocationID)
	rp.AllocateRequest(sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       1,
		ResourcePool:      "default",
	})

	require.Empty(t, rp.GetAllocationSummary(allocationID).Resources)
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID).Resources, 1)

	allocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.NotNil(t, allocated)
	require.Len(t, allocated.Resources, 1)

	for _, res := range allocated.Resources {
		conf := expconf.ExperimentConfig{ //nolint:exhaustruct
			RawEnvironment: &expconf.EnvironmentConfigV0{ //nolint:exhaustruct
				RawImage: &expconf.EnvironmentImageMapV0{ //nolint:exhaustruct
					RawCPU: ptrs.Ptr("ubuntu:latest"),
				},
				RawPodSpec: &expconf.PodSpec{
					// Make them unschedulable.
					Spec: k8sV1.PodSpec{NodeSelector: map[string]string{"non-existent": uuid.NewString()}},
				},
			},
		}
		conf = schemas.WithDefaults(conf)

		err := res.Start(nil, tasks.TaskSpec{
			Description:     fmt.Sprintf("test-job-%s", uuid.NewString()[:8]),
			Entrypoint:      []string{"sleep", "99999"},
			AgentUserGroup:  &model.AgentUserGroup{},
			Environment:     conf.Environment(),
			ResourcesConfig: conf.Resources(),
			DontShipLogs:    true,
		}, sproto.ResourcesRuntimeInfo{})
		defer res.Kill(nil)
		require.NoError(t, err)
	}

	ctxWaitForStarting, cancelWaitForStarting := context.WithTimeout(ctx, 5*time.Second)
	defer cancelWaitForStarting()
	for {
		ev, err := sub.GetWithContext(ctxWaitForStarting)
		if err != nil && errors.Is(err, context.DeadlineExceeded) {
			break
		} else if err != nil {
			t.Error(err)
			t.FailNow()
		}

		_, ok := ev.(*sproto.ResourcesStateChanged)
		if ok {
			t.Error("job should've stayed queued")
			t.FailNow()
		}
	}

	podListOpts := metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", allocationIDLabel, string(allocationID)),
	}
	pods, err := j.clientSet.CoreV1().Pods("default").List(ctx, podListOpts)
	require.NoError(t, err)

	require.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	err = j.clientSet.CoreV1().Pods("default").Delete(ctx, pod.Name, metaV1.DeleteOptions{})
	require.NoError(t, err)

	var stop *sproto.ResourcesStopped
	for state := sproto.Assigned; state != sproto.Terminated; {
		change := poll[*sproto.ResourcesStateChanged](ctx, t, sub)
		require.True(t, state.BeforeOrEqual(change.ResourcesState))
		state = change.ResourcesState
		stop = change.ResourcesStopped
	}
	require.NotNil(t, stop.Failure)
	require.Contains(t, stop.Failure.ErrMsg, "deleted pod")
}

func TestExternalPodDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	id := uuid.NewString()
	jobID, taskID, allocationID := model.JobID(id), model.TaskID(id), model.AllocationID(id)
	startTime := time.Now()

	err := tasklist.GroupPriorityChangeRegistry.Add(jobID, func(i int) error { return nil })
	require.NoError(t, err)

	sub := rmevents.Subscribe(allocationID)
	rp.AllocateRequest(sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       1,
		ResourcePool:      "default",
	})

	require.Empty(t, rp.GetAllocationSummary(allocationID).Resources)
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID).Resources, 1)

	allocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.NotNil(t, allocated)
	require.Len(t, allocated.Resources, 1)

	for _, res := range allocated.Resources {
		conf := expconf.ExperimentConfig{ //nolint:exhaustruct
			RawEnvironment: &expconf.EnvironmentConfigV0{ //nolint:exhaustruct
				RawImage: &expconf.EnvironmentImageMapV0{ //nolint:exhaustruct
					RawCPU: ptrs.Ptr("ubuntu:latest"),
				},
			},
		}
		conf = schemas.WithDefaults(conf)

		err := res.Start(nil, tasks.TaskSpec{
			Description:     fmt.Sprintf("test-job-%s", uuid.NewString()[:8]),
			Entrypoint:      []string{"sleep", "99999"},
			AgentUserGroup:  &model.AgentUserGroup{},
			Environment:     conf.Environment(),
			ResourcesConfig: conf.Resources(),
			DontShipLogs:    true,
		}, sproto.ResourcesRuntimeInfo{})
		defer res.Kill(nil)
		require.NoError(t, err)
	}

	change := poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Pulling, change.ResourcesState)

	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Starting, change.ResourcesState)

	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Running, change.ResourcesState)
	require.NotNil(t, change.ResourcesStarted)

	podListOpts := metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", allocationIDLabel, string(allocationID)),
	}
	pods, err := j.clientSet.CoreV1().Pods("default").List(ctx, podListOpts)
	require.NoError(t, err)

	require.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	err = j.clientSet.CoreV1().Pods("default").Delete(ctx, pod.Name, metaV1.DeleteOptions{})
	require.NoError(t, err)

	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Terminated, change.ResourcesState)
	require.NotNil(t, change.ResourcesStopped)
	require.NotNil(t, change.ResourcesStopped.Failure)
}

func TestReattach(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	user := db.RequireMockUser(t, db.SingleDB())
	task := db.RequireMockTask(t, db.SingleDB(), &user.ID)
	alloc := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)
	allocationID, taskID, jobID := alloc.AllocationID, task.TaskID, *task.JobID
	startTime := task.StartTime

	err := tasklist.GroupPriorityChangeRegistry.Add(jobID, func(i int) error { return nil })
	require.NoError(t, err)

	sub := rmevents.Subscribe(allocationID)
	allocateReq := sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       1,
		ResourcePool:      "default",
	}
	rp.AllocateRequest(allocateReq)

	require.Empty(t, rp.GetAllocationSummary(allocationID).Resources)
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID).Resources, 1)

	allocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.NotNil(t, allocated)
	require.Len(t, allocated.Resources, 1)

	for _, res := range allocated.Resources {
		conf := expconf.ExperimentConfig{ //nolint:exhaustruct
			RawEnvironment: &expconf.EnvironmentConfigV0{ //nolint:exhaustruct
				RawImage: &expconf.EnvironmentImageMapV0{ //nolint:exhaustruct
					RawCPU: ptrs.Ptr("ubuntu:latest"),
				},
			},
		}
		conf = schemas.WithDefaults(conf)

		err := res.Start(nil, tasks.TaskSpec{
			Description:     fmt.Sprintf("test-job-%s", uuid.NewString()[:8]),
			Entrypoint:      []string{"sleep", "99999"},
			AgentUserGroup:  &model.AgentUserGroup{},
			Environment:     conf.Environment(),
			ResourcesConfig: conf.Resources(),
			DontShipLogs:    true,
		}, sproto.ResourcesRuntimeInfo{})
		defer res.Kill(nil)
		require.NoError(t, err)
	}

	change := poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Pulling, change.ResourcesState)

	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Starting, change.ResourcesState)

	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Running, change.ResourcesState)
	require.NotNil(t, change.ResourcesStarted)

	// Remake all components and "reattach" to this new resource pool. This saves
	// us from needing to made the k8s code do graceful shutdown, but we should
	// do it anyway someday.
	rp = newTestResourcePool(newTestJobsService(t))

	sub = rmevents.Subscribe(allocationID)
	allocateReq.Restore = true
	rp.AllocateRequest(allocateReq)

	require.Empty(t, rp.GetAllocationSummary(allocationID).Resources)
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID).Resources, 1)

	reallocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.True(t, reallocated.Recovered)
	require.Len(t, reallocated.Resources, 1)

	for _, res := range reallocated.Resources {
		res.Kill(nil)
	}

	for state := sproto.Assigned; state != sproto.Terminated; {
		change := poll[*sproto.ResourcesStateChanged](ctx, t, sub)
		require.True(t, state.BeforeOrEqual(change.ResourcesState))
		state = change.ResourcesState
	}
}

func TestJobQueueReattach(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	user := db.RequireMockUser(t, db.SingleDB())
	task := db.RequireMockTask(t, db.SingleDB(), &user.ID)
	alloc := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)
	allocationID, taskID, jobID := alloc.AllocationID, task.TaskID, *task.JobID
	startTime := task.StartTime

	err := tasklist.GroupPriorityChangeRegistry.Add(jobID, func(i int) error { return nil })
	require.NoError(t, err)

	sub := rmevents.Subscribe(allocationID)
	allocateReq := sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       2,
		ResourcePool:      "default",
	}
	rp.AllocateRequest(allocateReq)
	rp.Admit()

	jobq := rp.GetJobQ()
	require.Len(t, jobq, 1)
	var info *sproto.RMJobInfo
	for _, tmp := range jobq {
		info = tmp
		break
	}
	require.Equal(t, sproto.SchedulingStateQueued, info.State)
	require.Equal(t, 0, info.AllocatedSlots)
	require.Equal(t, 2, info.RequestedSlots)

	allocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.NotNil(t, allocated)
	require.Len(t, allocated.Resources, 1)
	for _, res := range allocated.Resources {
		conf := expconf.ExperimentConfig{ //nolint:exhaustruct
			RawEnvironment: &expconf.EnvironmentConfigV0{ //nolint:exhaustruct
				RawImage: &expconf.EnvironmentImageMapV0{ //nolint:exhaustruct
					RawCPU: ptrs.Ptr("ubuntu:latest"),
				},
			},
		}
		conf = schemas.WithDefaults(conf)

		err := res.Start(nil, tasks.TaskSpec{
			Description:     fmt.Sprintf("test-job-%s", uuid.NewString()[:8]),
			Entrypoint:      []string{"sleep", "99999"},
			AgentUserGroup:  &model.AgentUserGroup{},
			Environment:     conf.Environment(),
			ResourcesConfig: conf.Resources(),
			DontShipLogs:    true,
		}, sproto.ResourcesRuntimeInfo{})
		defer res.Kill(nil)
		require.NoError(t, err)
	}
	change := poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Pulling, change.ResourcesState)
	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Starting, change.ResourcesState)
	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Running, change.ResourcesState)

	jobq = rp.GetJobQ()
	require.Len(t, jobq, 1)
	for _, tmp := range jobq {
		info = tmp
		break
	}
	require.Equal(t, sproto.SchedulingStateScheduled, info.State)
	require.Equal(t, 2, info.AllocatedSlots)
	require.Equal(t, 2, info.RequestedSlots)

	// Remake all components and "reattach" to this new resource pool. This saves
	// us from needing to made the k8s code do graceful shutdown, but we should
	// do it anyway someday.
	rp = newTestResourcePool(newTestJobsService(t))

	sub = rmevents.Subscribe(allocationID)
	allocateReq.Restore = true
	rp.AllocateRequest(allocateReq)
	rp.Admit()

	reallocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.True(t, reallocated.Recovered)
	require.Len(t, reallocated.Resources, 1)

	var reattachInfo *sproto.RMJobInfo
	require.True(t, waitForCondition(5*time.Second, func() bool {
		for _, i := range rp.GetJobQ() {
			reattachInfo = i
			return i.State == sproto.SchedulingStateScheduled
		}
		return false
	}), "job isn' showing scheduling within 5s of being reattached")
	require.Equal(t, 2, reattachInfo.AllocatedSlots)
	require.Equal(t, 2, reattachInfo.RequestedSlots)
}

func TestPartialJobsShowQueuedStates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	user := db.RequireMockUser(t, db.SingleDB())
	task := db.RequireMockTask(t, db.SingleDB(), &user.ID)
	alloc := db.RequireMockAllocation(t, db.SingleDB(), task.TaskID)
	allocationID, taskID, jobID := alloc.AllocationID, task.TaskID, *task.JobID
	startTime := task.StartTime

	err := tasklist.GroupPriorityChangeRegistry.Add(jobID, func(i int) error { return nil })
	require.NoError(t, err)

	var slots int
	agents, err := rp.jobsService.GetAgents()
	require.NoError(t, err)
	for _, n := range agents.Agents {
		slots += len(n.Slots)
	}

	sub := rmevents.Subscribe(allocationID)
	allocateReq := sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       2 * slots,
		ResourcePool:      "default",
	}
	rp.AllocateRequest(allocateReq)
	rp.Admit()

	allocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.NotNil(t, allocated)
	require.Len(t, allocated.Resources, 1)
	for _, res := range allocated.Resources {
		conf := expconf.ExperimentConfig{ //nolint:exhaustruct
			RawEnvironment: &expconf.EnvironmentConfigV0{ //nolint:exhaustruct
				RawImage: &expconf.EnvironmentImageMapV0{ //nolint:exhaustruct
					RawCPU: ptrs.Ptr("ubuntu:latest"),
				},
			},
		}
		conf = schemas.WithDefaults(conf)

		err := res.Start(nil, tasks.TaskSpec{
			Description:     fmt.Sprintf("test-job-%s", uuid.NewString()[:8]),
			Entrypoint:      []string{"sleep", "99999"},
			AgentUserGroup:  &model.AgentUserGroup{},
			Environment:     conf.Environment(),
			ResourcesConfig: conf.Resources(),
			DontShipLogs:    true,
		}, sproto.ResourcesRuntimeInfo{})
		defer res.Kill(nil)
		require.NoError(t, err)
	}

	shortCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	for {
		ev, err := sub.GetWithContext(shortCtx)
		if err != nil {
			break
		}

		res, ok := ev.(*sproto.ResourcesStateChanged)
		if !ok {
			continue
		}
		if sproto.Pulling.BeforeOrEqual(res.ResourcesState) {
			continue
		}
		t.Error("state went to PULLING or beyond when all pods could not have been scheduled")
		t.FailNow()
	}
}

func TestNodeWorkflows(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	resp, err := j.getAgents()
	require.NoError(t, err)
	require.Len(t, resp.Agents, 1)
	nodeID := resp.Agents[0].Id

	_, err = rp.jobsService.DisableAgent(&apiv1.DisableAgentRequest{AgentId: nodeID})
	defer func() {
		// Ensure we re-enable the agent, otherwise failures in this test will break others.
		_, err := rp.jobsService.EnableAgent(&apiv1.EnableAgentRequest{AgentId: nodeID})
		require.NoError(t, err)
	}()
	require.NoError(t, err)

	// Wait because this check relies on our informers (eventual consistency).
	require.True(t, waitForCondition(10*time.Second, func() bool {
		// Bust the cache. Calls that	 mutate nodes should probably handle this.
		j.mu.Lock()
		j.getAgentsCacheTime = j.getAgentsCacheTime.Add(-time.Hour)
		j.mu.Unlock()

		resp, err := j.GetAgents()
		require.NoError(t, err)
		require.Len(t, resp.Agents, 1)
		return !resp.Agents[0].Enabled
	}), "GetAgents didn't say %s is disabled, but we just disabled it", nodeID)

	id := uuid.NewString()
	jobID, taskID, allocationID := model.JobID(id), model.TaskID(id), model.AllocationID(id)
	startTime := time.Now()

	err = tasklist.GroupPriorityChangeRegistry.Add(jobID, func(i int) error { return nil })
	require.NoError(t, err)

	sub := rmevents.Subscribe(allocationID)
	rp.AllocateRequest(sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       1,
		ResourcePool:      "default",
	})

	require.Empty(t, rp.GetAllocationSummary(allocationID).Resources)
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID).Resources, 1)

	allocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.NotNil(t, allocated)
	require.Len(t, allocated.Resources, 1)

	for _, res := range allocated.Resources {
		conf := expconf.ExperimentConfig{ //nolint:exhaustruct
			RawEnvironment: &expconf.EnvironmentConfigV0{ //nolint:exhaustruct
				RawImage: &expconf.EnvironmentImageMapV0{ //nolint:exhaustruct
					RawCPU: ptrs.Ptr("ubuntu:latest"),
				},
			},
		}
		conf = schemas.WithDefaults(conf)

		err := res.Start(nil, tasks.TaskSpec{
			Description:     fmt.Sprintf("test-job-%s", uuid.NewString()[:8]),
			Entrypoint:      []string{"/bin/bash", "-c", "exit 0"},
			AgentUserGroup:  &model.AgentUserGroup{},
			Environment:     conf.Environment(),
			ResourcesConfig: conf.Resources(),
			DontShipLogs:    true,
		}, sproto.ResourcesRuntimeInfo{})
		defer res.Kill(nil)
		require.NoError(t, err)
	}

	shortCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		ev, err := sub.GetWithContext(shortCtx)
		if err != nil {
			break
		}

		res, ok := ev.(*sproto.ResourcesStateChanged)
		if !ok {
			continue
		}
		if res.ResourcesState.BeforeOrEqual(sproto.Pulling) {
			continue
		}
		t.Error("state went to RUNNING or beyond when all agents were disabled")
		t.FailNow()
	}

	_, err = rp.jobsService.EnableAgent(&apiv1.EnableAgentRequest{AgentId: nodeID})
	require.NoError(t, err)

	// Be careful to allow missing state changes here since the jobs are very short. It's
	// all good as long as we don't go backwards and end terminated.
	var stop *sproto.ResourcesStopped
	for state := sproto.Assigned; state != sproto.Terminated; {
		change := poll[*sproto.ResourcesStateChanged](ctx, t, sub)
		require.True(t, state.BeforeOrEqual(change.ResourcesState))
		state = change.ResourcesState
		stop = change.ResourcesStopped
	}
	require.NotNil(t, stop)
	require.Nil(t, stop.Failure)
}

func TestAllocateAndReleaseBeforeStarted(t *testing.T) {
	rp := newTestResourcePool(newTestJobsService(t))
	allocID := model.AllocationID(uuid.NewString())

	allocReq := sproto.AllocateRequest{
		AllocationID: allocID,
		JobID:        model.NewJobID(),
		Name:         uuid.NewString(),
	}
	rp.AllocateRequest(allocReq)
	summary := rp.GetAllocationSummary(allocID)
	require.NotNil(t, summary)
	require.Equal(t, allocReq.Name, summary.Name)

	rp.ResourcesReleased(sproto.ResourcesReleased{
		AllocationID: allocID,
		ResourcePool: rp.poolConfig.PoolName,
	})
	summary = rp.GetAllocationSummary(allocID)
	require.Nil(t, summary)
}

func TestGroupMaxSlots(t *testing.T) {
	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	id := uuid.NewString()
	jobID := model.JobID(id)

	err := tasklist.GroupPriorityChangeRegistry.Add(jobID, func(i int) error { return nil })
	require.NoError(t, err)

	t.Log("set group to have a max of one slot")
	rp.SetGroupMaxSlots(sproto.SetGroupMaxSlots{
		MaxSlots: ptrs.Ptr(1),
		JobID:    jobID,
	})

	t.Log("first one slot task in the job should get scheduled")
	taskID, allocationID := model.TaskID(id), model.AllocationID(id)
	startTime := time.Now()
	rp.AllocateRequest(sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       1,
		ResourcePool:      "default",
	})
	require.Empty(t, rp.GetAllocationSummary(allocationID).Resources)
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID).Resources, 1)

	t.Log("but the second shouldn't")
	id2 := uuid.NewString()
	taskID2, allocationID2 := model.TaskID(id2), model.AllocationID(id2)
	rp.AllocateRequest(sproto.AllocateRequest{
		AllocationID:      allocationID2,
		TaskID:            taskID2,
		JobID:             jobID,
		RequestTime:       startTime,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              "test job",
		SlotsNeeded:       1,
		ResourcePool:      "default",
	})
	require.Empty(t, rp.GetAllocationSummary(allocationID2).Resources)
	rp.Admit()
	require.Empty(t, rp.GetAllocationSummary(allocationID2).Resources)

	t.Log("and when the first releases it should get scheduled")
	rp.ResourcesReleased(sproto.ResourcesReleased{AllocationID: allocationID})
	rp.Admit()
	require.Len(t, rp.GetAllocationSummary(allocationID2).Resources, 1)
}

func TestPendingPreemption(t *testing.T) {
	var rp kubernetesResourcePool
	err := rp.PendingPreemption(sproto.PendingPreemption{})
	require.Equal(t, rmerrors.ErrNotSupported, err)
}

func TestSetGroupWeight(t *testing.T) {
	var rp kubernetesResourcePool
	err := rp.SetGroupWeight(sproto.SetGroupWeight{})
	require.Equal(t, rmerrors.UnsupportedError("set group weight is unsupported in k8s"), err)
}

func TestSetGroupPriority(t *testing.T) {
	rp := newTestResourcePool(newTestJobsService(t))

	cases := []struct {
		name        string
		newPriority int
		preemptible bool
	}{
		{"not-preemptible", 0, false},
		{"no change", int(config.KubernetesDefaultPriority), true},
		{"increase", 100, true},
		{"decrease", 1, true},
		{"negative", -10, true}, // doesn't make sense, but it is allowed
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			jobID := model.NewJobID()

			rp.AllocateRequest(sproto.AllocateRequest{
				JobID:       jobID,
				Preemptible: tt.preemptible,
			})

			err := rp.SetGroupPriority(sproto.SetGroupPriority{
				Priority:     tt.newPriority,
				ResourcePool: rp.poolConfig.PoolName,
				JobID:        jobID,
			})

			if tt.preemptible {
				require.NoError(t, err)
				// TODO (bradley): check that the priority change is reflected in rm events
				// require.Equal(t, tt.newPriority, *rp.getOrCreateGroup(jobID).Priority)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestValidateResources(t *testing.T) {
	rp := newTestResourcePool(newTestJobsService(t))

	cases := []struct {
		name           string
		slots          int
		maxSlotsPerPod int
		fulfillable    bool
	}{
		{"valid", 1, 2, true},
		{"invalid, not divisible", 10, 3, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rp.maxSlotsPerPod = tt.maxSlotsPerPod
			res := rp.ValidateResources(sproto.ValidateResourcesRequest{
				Slots: tt.slots,
			})
			if tt.fulfillable {
				require.NoError(t, res)
			} else {
				require.ErrorContains(t, res, "invalid resource request")
			}
		})
	}
}

func poll[T sproto.ResourcesEvent](ctx context.Context, t *testing.T, sub *sproto.ResourcesSubscription) T {
	for {
		ev, err := sub.GetWithContext(ctx)
		if err != nil {
			var typed T
			t.Errorf("failed to receive %T in time: %s", typed, err)
			t.Error(string(debug.Stack()))
			t.FailNow()
		}

		res, ok := ev.(T)
		if !ok {
			continue
		}
		return res
	}
}

var tickInterval = 10 * time.Millisecond

func waitForCondition(timeout time.Duration, condition func() bool) bool {
	for i := 0; i < int(timeout/tickInterval); i++ {
		if condition() {
			return true
		}
		time.Sleep(tickInterval)
	}
	return false
}

var testResourcePoolConfig = config.ResourcePoolConfig{
	PoolName:                 defaultResourcePool,
	Description:              "default test pool",
	TaskContainerDefaults:    &model.TaskContainerDefaultsConfig{},
	AgentReattachEnabled:     false,
	AgentReconnectWait:       0,
	KubernetesNamespace:      "default",
	MaxCPUContainersPerAgent: 0,
}

func newTestResourcePool(j *jobsService) *kubernetesResourcePool {
	rp := newResourcePool(1, &testResourcePoolConfig, j, db.SingleDB())
	j.jobSchedulingStateCallback = rp.JobSchedulingStateChanged
	return rp
}

func newTestJobsService(t *testing.T) *jobsService {
	j, err := newJobsService(
		"default",
		map[string]string{"default": defaultResourcePool},
		"",
		model.TLSClientConfig{},
		"",
		device.CPU,
		config.PodSlotResourceRequests{
			CPU: 1,
		},
		[]config.ResourcePoolConfig{
			testResourcePoolConfig,
		},
		&model.TaskContainerDefaultsConfig{},
		"localhost",
		8080,
		"",
		"~/.kube/config",
		nil,
		nil,
	)
	require.NoError(t, err)
	return j
}

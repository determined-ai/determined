package kubernetesrm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
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

func TestLaunch(t *testing.T) {
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

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocationID))
	rp.Schedule()
	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocationID))

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
			Entrypoint:      []string{"sleep", "1"},
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

	change = poll[*sproto.ResourcesStateChanged](ctx, t, sub)
	require.Equal(t, sproto.Terminated, change.ResourcesState)
	require.NotNil(t, change.ResourcesStopped)
	require.Nil(t, change.ResourcesStopped.Failure)
}

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
				ErrMsg:      "unrecoverable image pull errors in pod",
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

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocationID))
	rp.Schedule()
	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocationID))

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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocationID))
	rp.Schedule()
	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocationID))

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

	// Remake all component and "reattach" to this new resource pool. This saves
	// us from needing to made the k8s code do graceful shutdown, but we should
	// do it anyway someday.
	rp = newTestResourcePool(newTestJobsService(t))

	sub = rmevents.Subscribe(allocationID)
	allocateReq.Restore = true
	rp.AllocateRequest(allocateReq)

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocationID))
	rp.Schedule()
	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocationID))

	reallocated := poll[*sproto.ResourcesAllocated](ctx, t, sub)
	require.True(t, reallocated.Recovered)
	require.Len(t, reallocated.Resources, 1)

	seen := 0 // HACK: Because we don't have graceful shutdown, we have two log streamers up and get two events.
	for {
		log := poll[*sproto.ContainerLog](ctx, t, sub)
		if strings.Contains(log.Message(), secret) {
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

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocationID))
	rp.Schedule()
	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocationID))

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

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocationID))
	rp.Schedule()
	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocationID))

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

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocationID))
	rp.Schedule()
	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocationID))

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
			continue
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
	require.Contains(t, stop.Failure.ErrMsg, "unable to get exit code")
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

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocationID))
	rp.Schedule()
	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocationID))

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

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocationID))
	rp.Schedule()
	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocationID))

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

	// Remake all component and "reattach" to this new resource pool. This saves
	// us from needing to made the k8s code do graceful shutdown, but we should
	// do it anyway someday.
	rp = newTestResourcePool(newTestJobsService(t))

	sub = rmevents.Subscribe(allocationID)
	allocateReq.Restore = true
	rp.AllocateRequest(allocateReq)

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocationID))
	rp.Schedule()
	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocationID))

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

func TestNodeWorkflows(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	j := newTestJobsService(t)
	rp := newTestResourcePool(j)

	resp := j.getAgents()
	require.Equal(t, 1, len(resp.Agents))
	nodeID := resp.Agents[0].Id

	_, err := rp.jobsService.DisableAgent(&apiv1.DisableAgentRequest{AgentId: nodeID})
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

		resp = j.GetAgents()
		require.Equal(t, 1, len(resp.Agents))
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

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocationID))
	rp.Schedule()
	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocationID))

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

func poll[T sproto.ResourcesEvent](ctx context.Context, t *testing.T, sub *sproto.ResourcesSubscription) T {
	for {
		ev, err := sub.GetWithContext(ctx)
		if err != nil {
			var typed T
			t.Errorf("failed to receive %T in time: %s", typed, err)
			t.FailNow()
		}

		res, ok := ev.(T)
		if !ok {
			continue
		}
		return res
	}
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
	return newResourcePool(1, &testResourcePoolConfig, j, db.SingleDB())
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
		"~/.kube/config",
		nil,
	)
	require.NoError(t, err)
	return j
}

func TestGetNonDetPods(t *testing.T) {
	hiddenPods := []k8sV1.Pod{
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "no node name",
			},
		},
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name:   "has det label",
				Labels: map[string]string{determinedLabel: "t"},
			},
		},
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name:   "has det system label",
				Labels: map[string]string{determinedSystemLabel: "f"},
			},
		},
	}
	expectedPods := []k8sV1.Pod{
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "ns1",
			},
			Spec: k8sV1.PodSpec{
				NodeName: "a",
			},
		},
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "ns2",
			},
			Spec: k8sV1.PodSpec{
				NodeName: "a",
			},
		},
	}

	ns1 := &mocks.PodInterface{}
	ns1.On("List", mock.Anything, mock.Anything).Once().
		Return(&k8sV1.PodList{Items: append(hiddenPods, expectedPods[0])}, nil)

	ns2 := &mocks.PodInterface{}
	ns2.On("List", mock.Anything, mock.Anything).Once().
		Return(&k8sV1.PodList{Items: append(hiddenPods, expectedPods[1])}, nil)

	p := jobsService{
		podInterfaces: map[string]typedV1.PodInterface{
			"ns1": ns1,
			"ns2": ns2,
		},
	}

	actualPods, err := p.getNonDetPods()
	require.NoError(t, err)
	require.ElementsMatch(t, expectedPods, actualPods)
}

func TestTaintTolerated(t *testing.T) {
	cases := []struct {
		expected    bool
		taint       k8sV1.Taint
		tolerations []k8sV1.Toleration
	}{
		{
			expected: true,
			taint:    taintFooBar,
			tolerations: []k8sV1.Toleration{{
				Key:      taintFooBar.Key,
				Value:    taintFooBar.Value,
				Operator: k8sV1.TolerationOpEqual,
			}},
		}, {
			expected: true,
			taint:    taintFooBar,
			tolerations: []k8sV1.Toleration{{
				Key:      taintFooBar.Key,
				Operator: k8sV1.TolerationOpExists,
			}},
		}, {
			expected: true,
			taint:    taintFooBar,
			tolerations: []k8sV1.Toleration{
				{
					Key:      taintFooBar.Key,
					Value:    taintFooBar.Value,
					Operator: k8sV1.TolerationOpEqual,
				}, {
					Key:      "baz",
					Value:    "qux",
					Operator: k8sV1.TolerationOpEqual,
				},
			},
		}, {
			expected: false,
			taint:    taintFooBar,
			tolerations: []k8sV1.Toleration{{
				Key:      taintFooBar.Key,
				Value:    taintFooBar.Value + taintFooBar.Value,
				Operator: k8sV1.TolerationOpEqual,
			}},
		}, {
			expected:    false,
			taint:       taintFooBar,
			tolerations: []k8sV1.Toleration{},
		}, {
			expected:    false,
			taint:       taintFooBar,
			tolerations: nil,
		},
	}

	for i, c := range cases {
		actual := taintTolerated(c.taint, c.tolerations)
		require.Equal(t, c.expected, actual, "test case %d failed", i)
	}
}

func TestAllTaintsTolerated(t *testing.T) {
	cases := []struct {
		expected    bool
		taints      []k8sV1.Taint
		tolerations []k8sV1.Toleration
	}{
		{
			expected:    true,
			taints:      nil,
			tolerations: nil,
		}, {
			expected: true,
			taints:   nil,
			tolerations: []k8sV1.Toleration{
				{
					Key:      taintFooBar.Key,
					Value:    taintFooBar.Value,
					Operator: k8sV1.TolerationOpEqual,
				},
			},
		}, {
			expected:    false,
			taints:      []k8sV1.Taint{taintFooBar},
			tolerations: nil,
		},
	}

	for i, c := range cases {
		actual := allTaintsTolerated(c.taints, c.tolerations)
		require.Equal(t, c.expected, actual, "test case %d failed", i)
	}
}

var taintFooBar = k8sV1.Taint{
	Key:    "foo",
	Value:  "bar",
	Effect: k8sV1.TaintEffectNoSchedule,
}

const fakeKubeconfig = `
apiVersion: v1
clusters:
- cluster:
    extensions:
    - extension:
        last-update: Mon, 04 Mar 2024 18:53:00 EST
        provider: minikube.sigs.k8s.io
        version: v1.29.0
      name: cluster_info
    server: https://127.0.0.1:49216
  name: minikube
contexts:
- context:
    cluster: minikube
    extensions:
    - extension:
        last-update: Mon, 04 Mar 2024 18:53:00 EST
        provider: minikube.sigs.k8s.io
        version: v1.29.0
      name: context_info
    namespace: default
    user: minikube
  name: minikube
current-context: minikube
kind: Config
preferences: {}
`

func Test_readClientConfig(t *testing.T) {
	customPath := "test_kube.config"
	err := os.WriteFile(customPath, []byte(fakeKubeconfig), 0o600)
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(customPath); err != nil {
			t.Logf("failed to cleanup %s", err)
		}
	}()

	tests := []struct {
		name           string
		kubeconfigPath string
		want           string
	}{
		{
			name:           "fallback to in cluster config",
			kubeconfigPath: "",
			want:           "unable to load in-cluster configuration",
		},
		{
			name:           "custom kubeconfig",
			kubeconfigPath: customPath,
			want:           "",
		},
		{
			name:           "custom kubeconfig with homedir expansion at least tried the correct file",
			kubeconfigPath: "~",
			want:           "is a directory", // Bit clever, but we're sure we expanded it with this error.
		},
		{
			name:           "this test can actually fail",
			kubeconfigPath: "a_file_that_doesn't_exist.config",
			want:           "no such file or",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readClientConfig(tt.kubeconfigPath)
			if tt.want != "" {
				require.ErrorContains(t, err, tt.want)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

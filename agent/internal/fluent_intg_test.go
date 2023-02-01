//go:build integration
// +build integration

package internal

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"sort"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/test/testutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestFluentPostgresLogging(t *testing.T) {
	// GIVEN a mock master that accepts logs
	logBuffer, cl := mockLogAcceptor(t)
	defer cl()

	// AND a successfully started fluent actor
	aConf := defaultAgentConfig()
	f, err := newFluentActor(aConf, aproto.MasterSetAgentOptions{
		MasterInfo: aproto.MasterInfo{},
		LoggingOptions: model.LoggingConfig{
			DefaultLoggingConfig: &model.DefaultLoggingConfig{},
		},
	})
	assert.NilError(t, err, "error starting fluentbit")
	sys := actor.NewSystem("")
	sys.MustActorOf(actor.Addr("fluent"), f)

	// WHEN a container prints some predefined logs
	taskID := model.NewTaskID()
	expected, actual := makeLogTestCase(taskID, aConf.AgentID)
	runContainerWithLogs(t, actual, taskID, f.port)

	// THEN fluent should parse all fields as expected and ship them to the mock master.
	var logs []model.TaskLog
	for i := 0; i < len(expected); i++ {
		select {
		case l := <-logBuffer:
			logs = append(logs, l)
		case <-time.After(time.Minute):
			assert.Equal(t, i, len(expected), "not enough logs received after one minute")
		}
	}
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.Before(*logs[j].Timestamp)
	})
	for i := range logs {
		assertLogEquals(t, logs[i], expected[i])
	}
}

func TestFluentLoggingElastic(t *testing.T) {
	// GIVEN an elastic instance to accept logs
	elastic, err := testutils.ResolveElastic()
	assert.NilError(t, err, "unable to connect to master")

	assert.NilError(t, elastic.AddDateNanosTemplate(), "unable to add template")

	// AND a successfully started fluent actor
	aConf := defaultAgentConfig()
	f, err := newFluentActor(aConf, aproto.MasterSetAgentOptions{
		MasterInfo:     aproto.MasterInfo{},
		LoggingOptions: testutils.DefaultElasticConfig(),
	})
	assert.NilError(t, err, "error starting fluentbit")
	sys := actor.NewSystem("")
	sys.MustActorOf(actor.Addr("fluent"), f)

	// WHEN a container prints some predefined logs
	taskID := model.NewTaskID()
	expected, actual := makeLogTestCase(taskID, aConf.AgentID)
	runContainerWithLogs(t, actual, taskID, f.port)

	// This is really unfortunate, but we don't query for logs until they're more than 10 seconds old
	// to _try_ to avoid the trickiness involved with elastic's consistency model.
	time.Sleep(11 * time.Second)

	assert.NilError(t, elastic.WaitForIngest(testutils.CurrentLogstashElasticIndex()))

	// THEN fluent should parse all fields as expected and ship them to elastic.
	logs, _, err := elastic.TaskLogs(taskID, 4, nil, apiv1.OrderBy_ORDER_BY_ASC, nil)
	assert.NilError(t, err, "failed to retrieve task logs")
	assert.Equal(t, len(logs), len(expected), "not enough logs received after one minute")

	for i := 0; i < 4; i++ {
		t.Logf("expected[%d] \n%+v actual[%d] \n%+v", i, expected[i], i, logs[i])
	}

	for i, l := range logs {
		assertLogEquals(t, *l, expected[i])
	}
}

func makeLogTestCase(taskID model.TaskID, agentID string) ([]model.TaskLog, string) {
	expected := []model.TaskLog{
		{
			TaskID:       taskID.String(),
			AllocationID: taskToAllocationID(taskID.String()),
			AgentID:      &agentID,
			ContainerID:  ptrs.Ptr("goodcontainer"),
			RankID:       ptrs.Ptr(4),
			Log:          "\n",
			StdType:      ptrs.Ptr("stdout"),
		},
		{
			TaskID:       taskID.String(),
			AllocationID: taskToAllocationID(taskID.String()),
			AgentID:      &agentID,
			ContainerID:  ptrs.Ptr("goodcontainer"),
			RankID:       ptrs.Ptr(1),
			Level:        ptrs.Ptr("INFO"),
			Log:          "Workload completed: <RUN_STEP (100 Batches): (580,6289,4)> (duration 0:00:01.496132)\n", //nolint:lll // Anything to make these lines shorter would look worse.
			StdType:      ptrs.Ptr("stdout"),
		},
		{
			TaskID:       taskID.String(),
			AllocationID: taskToAllocationID(taskID.String()),
			AgentID:      &agentID,
			ContainerID:  ptrs.Ptr("goodcontainer"),
			RankID:       ptrs.Ptr(2),
			Level:        ptrs.Ptr("ERROR"),
			Log:          "Workload completed: <RUN_STEP (100 Batches): (580,6289,4)> (duration 9:99:99)\n", // nolint:lll
			StdType:      ptrs.Ptr("stdout"),
		},
		{
			TaskID:       taskID.String(),
			AllocationID: taskToAllocationID(taskID.String()),
			AgentID:      &agentID,
			ContainerID:  ptrs.Ptr("goodcontainer"),
			RankID:       ptrs.Ptr(3),
			Log:          "urllib3.exceptions.NewConnectionError: <urllib3.connection.HTTPConnection object at 0x7f29a414dd30>: Failed to establish a new connection: [Errno 110]\n", // nolint:lll
			StdType:      ptrs.Ptr("stdout"),
		},
	}
	actual := `[rank=4] 
[rank=1] INFO: Workload completed: <RUN_STEP (100 Batches): (580,6289,4)> (duration 0:00:01.496132)
[rank=2] ERROR: Workload completed: <RUN_STEP (100 Batches): (580,6289,4)> (duration 9:99:99)
[rank=3] urllib3.exceptions.NewConnectionError: <urllib3.connection.HTTPConnection object at 0x7f29a414dd30>: Failed to establish a new connection: [Errno 110]` // nolint:lll
	return expected, actual
}

func mockLogAcceptor(t *testing.T) (chan model.TaskLog, func()) {
	e := echo.New()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 8080))
	assert.NilError(t, err, "error starting mock master listener")
	e.Listener = lis
	logBuffer := make(chan model.TaskLog)
	e.POST("/task-logs", func(ctx echo.Context) error {
		body, err := ioutil.ReadAll(ctx.Request().Body)
		if err != nil {
			return err
		}

		var logs []model.TaskLog
		if err = json.Unmarshal(body, &logs); err != nil {
			return err
		}

		for _, l := range logs {
			logBuffer <- l
		}
		return nil
	})
	go func() {
		if err := e.StartServer(e.Server); err != nil {
			t.Logf("mock master exited: %s", err)
		}
	}()
	return logBuffer, func() {
		if err := e.Close(); err != nil {
			t.Logf("error closing mock master echo server: %s", err)
		}
		if err := e.Listener.Close(); err != nil {
			t.Logf("error closing mock master listener: %s", err)
		}
	}
}

func runContainerWithLogs(t *testing.T, fakeLogs string, taskID model.TaskID, fluentPort int) {
	// WHEN we start a container that logs with fluentbit as its log driver.
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.40"))
	assert.NilError(t, err, "error connecting to Docker daemon")

	logs, err := docker.ImagePull(context.Background(), "busybox", types.ImagePullOptions{})
	assert.NilError(t, err, "failed to pull image")
	_, err = ioutil.ReadAll(logs)
	assert.NilError(t, err, "failed to drain pull logs")

	// AND set some known ENV values.
	spec := cproto.Spec{
		RunSpec: cproto.RunSpec{
			ContainerConfig: container.Config{
				Cmd:   []string{"cat", "/fakelogs"},
				Image: "busybox",
			},
			UseFluentLogging: true,
		},
	}
	env := []string{
		fmt.Sprintf("DET_TASK_ID=%s", taskID),
		fmt.Sprintf("DET_ALLOCATION_ID=%s", *taskToAllocationID(taskID.String())),
	}
	cont := cproto.Container{ID: "goodcontainer"}
	spec, err = overwriteSpec(cont, spec, env, nil, fluentPort, true)
	assert.NilError(t, err, "failed to overwrite spec")

	// AND create a container with this env
	cc, err := docker.ContainerCreate(
		context.Background(),
		&spec.RunSpec.ContainerConfig,
		&spec.RunSpec.HostConfig,
		&spec.RunSpec.NetworkingConfig,
		nil,
		"log-test-"+uuid.New().String())
	assert.NilError(t, err, "error creating container")

	// AND start that container with a command to print the fake logs
	files := []archive.Item{
		{
			Path:     "/fakelogs",
			Type:     tar.TypeReg,
			FileMode: 0o444,
			Content:  []byte(fakeLogs),
		},
	}
	filesReader, err := archive.ToIOReader(files)
	assert.NilError(t, err, "failed make reader from fluent files")
	err = docker.CopyToContainer(context.Background(),
		cc.ID,
		"/",
		filesReader,
		types.CopyToContainerOptions{},
	)
	assert.NilError(t, err, "failed to copy files to container")
	err = docker.ContainerStart(context.Background(), cc.ID, types.ContainerStartOptions{})
	assert.NilError(t, err, "error starting container")

	exitChan, errChan := docker.ContainerWait(
		context.Background(), cc.ID, container.WaitConditionNextExit)
	select {
	case err = <-errChan:
		assert.NilError(t, err, "container wait failed")
	case exit := <-exitChan:
		if exit.Error != nil {
			t.Fatalf("container exited with error: %s", exit.Error)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("container did not exit after 30 seconds")
	}
}

func assertLogEquals(t *testing.T, actual, expected model.TaskLog) {
	// nil out timestamps since they are set by fluent and we cannot know what to expect.
	actual.Timestamp = nil
	// IDs are assigned unpredictably by the backend, so don't compare them.
	actual.ID = nil
	actual.StringID = nil
	assert.DeepEqual(t, actual, expected)
}

func taskToAllocationID(taskID string) *string {
	return ptrs.Ptr(taskID + ".0")
}

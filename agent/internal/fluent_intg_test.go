//+build integration

package internal

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net"
	"sort"
	"testing"
	"time"

	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/labstack/echo"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/archive"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/test/testutils"
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
	trialID := math.MaxInt16 + rand.Int31n(math.MaxInt16)
	expected, actual := makeLogTestCase(int(trialID), aConf.AgentID)
	runContainerWithLogs(t, actual, int(trialID), f.port)

	// THEN fluent should parse all fields as expected and ship them to the mock master.
	var logs []model.TrialLog
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
	trialID := math.MaxInt16 + rand.Int31n(math.MaxInt16)
	expected, actual := makeLogTestCase(int(trialID), aConf.AgentID)
	runContainerWithLogs(t, actual, int(trialID), f.port)

	// This is really unfortunate, but we don't query for logs until they're more than 10 seconds old
	// to _try_ to avoid the trickiness involved with elastic's consistency model.
	time.Sleep(11 * time.Second)

	assert.NilError(t, elastic.WaitForIngest(testutils.CurrentLogstashElasticIndex()))

	// THEN fluent should parse all fields as expected and ship them to elastic.
	logs, _, err := elastic.TrialLogs(
		int(trialID), 0, 4, nil, apiv1.OrderBy_ORDER_BY_ASC, nil)
	assert.NilError(t, err, "failed to retrieve trial logs")
	assert.Equal(t, len(logs), len(expected), "not enough logs received after one minute")
	for i, l := range logs {
		assertLogEquals(t, *l, expected[i])
	}
}

func makeLogTestCase(trialID int, agentID string) ([]model.TrialLog, string) {
	expected := []model.TrialLog{
		{
			TrialID:     trialID,
			AgentID:     &agentID,
			ContainerID: stringToPointer("goodcontainer"),
			RankID:      intToPointer(4),
			Log:         stringToPointer("\n"),
			StdType:     stringToPointer("stdout"),
		},
		{
			TrialID:     trialID,
			AgentID:     &agentID,
			ContainerID: stringToPointer("goodcontainer"),
			RankID:      intToPointer(1),
			Level:       stringToPointer("INFO"),
			Log:         stringToPointer("Workload completed: <RUN_STEP (100 Batches): (580,6289,4)> (duration 0:00:01.496132)\n"), //nolint:lll // Anything to make these lines shorter would look worse.
			StdType:     stringToPointer("stdout"),
		},
		{
			TrialID:     trialID,
			AgentID:     &agentID,
			ContainerID: stringToPointer("goodcontainer"),
			RankID:      intToPointer(2),
			Level:       stringToPointer("ERROR"),
			Log:         stringToPointer("Workload completed: <RUN_STEP (100 Batches): (580,6289,4)> (duration 9:99:99)\n"), // nolint:lll
			StdType:     stringToPointer("stdout"),
		},
		{
			TrialID:     trialID,
			AgentID:     &agentID,
			ContainerID: stringToPointer("goodcontainer"),
			RankID:      intToPointer(3),
			Log:         stringToPointer("urllib3.exceptions.NewConnectionError: <urllib3.connection.HTTPConnection object at 0x7f29a414dd30>: Failed to establish a new connection: [Errno 110]\n"), // nolint:lll
			StdType:     stringToPointer("stdout"),
		},
	}
	actual := `[rank=4] 
[rank=1] INFO: Workload completed: <RUN_STEP (100 Batches): (580,6289,4)> (duration 0:00:01.496132)
[rank=2] ERROR: Workload completed: <RUN_STEP (100 Batches): (580,6289,4)> (duration 9:99:99)
[rank=3] urllib3.exceptions.NewConnectionError: <urllib3.connection.HTTPConnection object at 0x7f29a414dd30>: Failed to establish a new connection: [Errno 110]` // nolint:lll
	return expected, actual
}

func mockLogAcceptor(t *testing.T) (chan model.TrialLog, func()) {
	e := echo.New()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 8080))
	assert.NilError(t, err, "error starting mock master listener")
	e.Listener = lis
	logBuffer := make(chan model.TrialLog)
	e.POST("/trial_logs", func(ctx echo.Context) error {
		body, err := ioutil.ReadAll(ctx.Request().Body)
		if err != nil {
			return err
		}

		var logs []model.TrialLog
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

func runContainerWithLogs(t *testing.T, fakeLogs string, trialID, fluentPort int) {
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
	env := []string{fmt.Sprintf("DET_TRIAL_ID=%d", trialID)}
	cont := cproto.Container{ID: "goodcontainer"}
	spec = overwriteSpec(cont, spec, env, nil, fluentPort)

	// AND create a container with this env
	cc, err := docker.ContainerCreate(
		context.Background(),
		&spec.RunSpec.ContainerConfig,
		&spec.RunSpec.HostConfig,
		&spec.RunSpec.NetworkingConfig,
		"log-test-"+uuid.New().String())
	assert.NilError(t, err, "error creating container")

	// AND start that container with a command to print the fake logs
	files := []archive.Item{
		{
			Path:     "/fakelogs",
			Type:     tar.TypeReg,
			FileMode: 0444,
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
	case <-time.After(10 * time.Second):
		t.Fatal("container did not exit after 10 seconds")
	}
}

func assertLogEquals(t *testing.T, l, expected model.TrialLog) {
	// nil out timestamps since they are set by fluent and we cannot know what to expect.
	l.Timestamp = nil
	// nil out the message since we don't really care, we just want to see the structured fields.
	l.Message = ""
	assert.DeepEqual(t, l, expected)
}

func stringToPointer(x string) *string {
	return &x
}

func intToPointer(x int) *int {
	return &x
}

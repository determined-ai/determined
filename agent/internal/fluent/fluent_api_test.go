//go:build integration

package fluent_test

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	dclient "github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/internal/fluent"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	atestutils "github.com/determined-ai/determined/agent/test/testutils"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/test/testutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestFluentPostgresLogging(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	ctx := context.Background()
	opts := atestutils.DefaultAgentConfig(0)
	mopts := atestutils.DefaultMasterSetAgentConfig()

	t.Log("building docker client")
	rawCl, err := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
	require.NoError(t, err)
	defer func() {
		if cErr := rawCl.Close(); cErr != nil {
			t.Logf("closing docker client: %s", cErr)
		}
	}()
	cl := docker.NewClient(rawCl)

	t.Log("building mock log acceptor")
	logBuffer, logsCloser := mockLogAcceptor(t, opts.MasterPort)
	defer logsCloser()

	t.Log("starting fluentbit")
	f, err := fluent.Start(ctx, opts, mopts, cl)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()

	t.Log("running container with predefined logs")
	taskID := model.NewTaskID()
	expected, expectedFile := makeLogTestCase(taskID, opts.AgentID)
	runContainerWithLogs(t, expectedFile, opts.AgentID, taskID, opts.Fluent.Port)

	t.Log("checking logs")
	var actual []model.TaskLog
	for i := 0; i < len(expected); i++ {
		select {
		case l := <-logBuffer:
			actual = append(actual, l)
		case <-time.After(10 * time.Second):
			require.Equal(t, i, len(expected), "not enough logs received after ten seconds")
		}
	}
	sort.Slice(actual, func(i, j int) bool {
		return actual[i].Timestamp.Before(*actual[j].Timestamp)
	})
	for i := range actual {
		require.True(t, assertLogEquals(t, actual[i], expected[i]))
	}
}

func TestFluentLoggingElastic(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	ctx := context.Background()
	opts := atestutils.DefaultAgentConfig(1)
	mopts := atestutils.ElasticMasterSetAgentConfig()

	t.Log("setting up elastic")
	elastic, err := testutils.ResolveElastic()
	require.NoError(t, err, "unable to connect to master")
	require.NoError(t, elastic.AddDateNanosTemplate(), "unable to add template")

	t.Log("building client")
	rawCl, err := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
	require.NoError(t, err)
	defer func() {
		if cErr := rawCl.Close(); cErr != nil {
			t.Logf("closing docker client: %s", cErr)
		}
	}()
	cl := docker.NewClient(rawCl)

	t.Log("starting fluentbit")
	f, err := fluent.Start(ctx, opts, mopts, cl)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close(), "close fluent failed")
	}()

	t.Log("running container with some predefined logs")
	taskID := model.NewTaskID()
	expected, actual := makeLogTestCase(taskID, opts.AgentID)
	runContainerWithLogs(t, actual, opts.AgentID, taskID, opts.Fluent.Port)

	t.Log("checking logs in elastic")
	for i := 0; i < 30; i++ {
		logs, _, err := elastic.TaskLogs(taskID, 4, nil, apiv1.OrderBy_ORDER_BY_ASC, nil)
		require.NoError(t, err, "failed to retrieve task logs")
		if len(logs) != len(expected) {
			t.Logf("checking logs again after delay... (%d, %d/%d found)", i, len(logs), len(expected))

			for i, l := range expected {
				j, err := json.MarshalIndent(l, "", "  ")
				require.NoError(t, err)
				t.Logf("expected[%d] = %s", i, j)
			}
			for i, l := range logs {
				j, err := json.MarshalIndent(l, "", "  ")
				require.NoError(t, err)
				t.Logf("actual[%d] = %s", i, j)
			}

			time.Sleep(time.Second)
			continue
		}

		expectedFound := true
		for i, l := range logs {
			expectedFound = expectedFound && assertLogEquals(t, *l, expected[i])
		}
		require.True(t, expectedFound, spew.Sdump(logs), spew.Sdump(expected))
		return
	}
	require.FailNow(t, "logs never showed up")
}

func TestFluentSadPaths(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	logC := atestutils.NewLogChannel(1024)
	log.AddHook(logC)
	ctx := context.Background()
	opts := atestutils.DefaultAgentConfig(2)
	mopts := atestutils.DefaultMasterSetAgentConfig()

	t.Log("building docker client")
	rawCl, err := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
	require.NoError(t, err)
	defer func() {
		if cErr := rawCl.Close(); cErr != nil {
			t.Logf("closing docker client: %s", cErr)
		}
	}()
	cl := docker.NewClient(rawCl)

	t.Log("_not_ starting log acceptor")
	t.Log("starting fluentbit")
	f, err := fluent.Start(ctx, opts, mopts, cl)
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()

	t.Log("running container with predefined logs")
	taskID := model.NewTaskID()
	_, expectedFile := makeLogTestCase(taskID, opts.AgentID)
	runContainerWithLogs(t, expectedFile, opts.AgentID, taskID, opts.Fluent.Port)

	t.Log("checking fluentbit failure logs for intermittent failure")
	found := false
	for e := range logC {
		if strings.Contains(e.Message, "failed to flush chunk") {
			found = true
			break
		}
	}
	require.Truef(t, found, "did not find failure log in output")

	t.Log("forcibly removing fluentbit")
	fs := filters.NewArgs()
	fs.Add("name", opts.Fluent.ContainerName)
	conts, err := rawCl.ContainerList(ctx, types.ContainerListOptions{
		Filters: fs,
	})
	require.NoError(t, err)
	for _, cont := range conts {
		err = cl.RemoveContainer(ctx, cont.ID, true)
		require.NoError(t, err)
	}
	err = f.Wait()
	require.Error(t, err)
	require.ErrorContains(t, err, "unexpected Fluent Bit exit (137)")

	t.Log("checking fluentbit failure logs for total failure")
	found = false
	for e := range logC {
		if strings.Contains(
			e.Message,
			"Fluent Bit logs ended unexpectedly",
		) {
			found = true
			break
		}
	}
	require.Truef(t, found, "did not find failure log in output")
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
			Log:          "Workload not complete: <RUN_STEP (100 Batches): (581,6290,5)> (duration 9:99:99)\n", // nolint:lll
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
	//nolint:lll
	actual := `[rank=4] 
[rank=1] INFO: Workload completed: <RUN_STEP (100 Batches): (580,6289,4)> (duration 0:00:01.496132)
[rank=2] ERROR: Workload not complete: <RUN_STEP (100 Batches): (581,6290,5)> (duration 9:99:99)
[rank=3] urllib3.exceptions.NewConnectionError: <urllib3.connection.HTTPConnection object at 0x7f29a414dd30>: Failed to establish a new connection: [Errno 110]
`
	return expected, actual
}

func mockLogAcceptor(t *testing.T, port int) (chan model.TaskLog, func()) {
	e := echo.New()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	require.NoError(t, err, "error starting mock master listener")
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
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := e.StartServer(e.Server); err != nil {
			t.Logf("mock master exited: %s", err)
		}
	}()
	return logBuffer, func() {
		if err := e.Close(); err != nil {
			t.Logf("error closing mock master echo server: %s", err)
		}
		wg.Wait()
	}
}

func runContainerWithLogs(
	t *testing.T, fakeLogs string, aID string, tID model.TaskID, fluentPort int,
) {
	rawCl, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.40"))
	require.NoError(t, err, "error connecting to Docker daemon")
	dCli := docker.NewClient(rawCl)

	cID := "goodcontainer"
	spec := cproto.Spec{
		RunSpec: cproto.RunSpec{
			ContainerConfig: dcontainer.Config{
				Cmd:   []string{"cat", "/fakelogs"},
				Image: "busybox",
				Env: []string{
					fmt.Sprintf("%s=%s", container.AgentIDEnvVar, aID),
					fmt.Sprintf("%s=%s", container.ContainerIDEnvVar, cID),
					fmt.Sprintf("%s=%s", container.TaskIDEnvVar, tID),
					fmt.Sprintf("%s=%s.%d", container.AllocationIDEnvVar, tID, 0),
				},
			},
			HostConfig: dcontainer.HostConfig{
				LogConfig: dcontainer.LogConfig{
					Type: "fluentd",
					Config: map[string]string{
						"fluentd-address":              "localhost:" + strconv.Itoa(fluentPort),
						"fluentd-sub-second-precision": "true",
						"mode":                         "non-blocking",
						"max-buffer-size":              "10m",
						"env":                          strings.Join(fluent.EnvVarNames, ","),
					},
				},
				AutoRemove: true,
			},
			UseFluentLogging: true,
			Archives: []cproto.RunArchive{
				{
					Path: "/",
					Archive: []archive.Item{
						{
							Path:     "/fakelogs",
							Type:     tar.TypeReg,
							Content:  []byte(fakeLogs),
							FileMode: 0o0777,
						},
					},
				},
			},
		},
	}

	c := container.Start(aproto.StartContainer{
		Container: cproto.Container{ID: cproto.ID(cID)},
		Spec:      spec,
	}, dCli, events.NilPublisher[container.Event]{})
	require.Nil(t, c.Wait().ContainerStopped.Failure)
	return
}

func assertLogEquals(t *testing.T, actual, expected model.TaskLog) bool {
	// nil out timestamps since they are set by fluent and we cannot know what to expect.
	actual.Timestamp = nil
	// IDs are assigned unpredictably by the backend, so don't compare them.
	actual.ID = nil
	actual.StringID = nil
	actualJSON, err := json.Marshal(actual)
	require.NoError(t, err)
	expectedJSON, err := json.Marshal(actual)
	require.NoError(t, err)
	return assert.JSONEq(t, string(expectedJSON), string(actualJSON))
}

func taskToAllocationID(taskID string) *string {
	return ptrs.Ptr(taskID + ".0")
}

//go:build integration

package docker_test

import (
	"archive/tar"
	"context"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dclient "github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
)

const (
	testImage = "determinedai/determined-agent:7c12bd2545e4e98018fa37f5326f377ff58583b2"
)

func TestPullImage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Log("building client")
	rawCl, err := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
	require.NoError(t, err)
	defer func() {
		if cErr := rawCl.Close(); cErr != nil {
			t.Logf("closing docker client: %s", cErr)
		}
	}()
	cl := docker.NewClient(rawCl)

	t.Log("removing image")
	switch _, err = rawCl.ImageRemove(ctx, testImage, types.ImageRemoveOptions{Force: true}); {
	case err == nil, strings.Contains(err.Error(), "No such image"):
		break
	case err != nil:
		t.Errorf("removing image: %s", err.Error())
		return
	}

	t.Log("normal pull, image not pulled")
	evs := make(chan docker.Event, 1024)
	pub := events.ChannelPublisher(evs)
	if err = cl.PullImage(ctx, docker.PullImage{Name: testImage}, pub); err != nil {
		t.Errorf("pulling image: %s", err.Error())
		return
	}
	close(evs)
	if !witnessedPull(t, evs) {
		t.Errorf("did not witness expected pull events")
		return
	}
	_, _, err = rawCl.ImageInspectWithRaw(ctx, testImage)
	require.NoError(t, err)

	t.Log("normal pull, image already pulled")
	evs = make(chan docker.Event, 64) // Excessively large value, so the client never blocks.
	pub = events.ChannelPublisher(evs)
	if err = cl.PullImage(ctx, docker.PullImage{Name: testImage}, pub); err != nil {
		t.Errorf("pulling image: %s", err.Error())
		return
	}
	close(evs)
	if witnessedPull(t, evs) {
		t.Error("saw pull of pulled image")
		return
	}
	_, _, err = rawCl.ImageInspectWithRaw(ctx, testImage)
	require.NoError(t, err)

	t.Log("force pull, image already pulled")
	evs = make(chan docker.Event, 64) // Excessively large value, so the client never blocks.
	pub = events.ChannelPublisher(evs)
	if err = cl.PullImage(ctx, docker.PullImage{
		Name:      testImage,
		ForcePull: true,
	}, pub); err != nil {
		t.Errorf("pulling image: %s", err.Error())
		return
	}
	close(evs)
	if !witnessedPull(t, evs) {
		t.Errorf("did not witness expected pull events")
		return
	}
	_, _, err = rawCl.ImageInspectWithRaw(ctx, testImage)
	require.NoError(t, err)
}

func witnessedPull(t *testing.T, events <-chan docker.Event) bool {
	pullWitnessed, statsBeginWitnessed, statsEndWitnessed := false, false, false
	for event := range events {
		switch {
		case event.Log != nil:
			switch log := event.Log; {
			case strings.Contains(log.Message, "pulling image"):
				pullWitnessed = true
			case strings.Contains(log.Message, "checking for updates"):
				pullWitnessed = true
			}
		case event.Stats != nil:
			switch stats := event.Stats; {
			case stats.Kind == docker.ImagePullStatsKind && stats.StartTime != nil:
				statsBeginWitnessed = true
			case stats.Kind == docker.ImagePullStatsKind && stats.EndTime != nil:
				statsEndWitnessed = true
			}
		}
	}
	return pullWitnessed && statsBeginWitnessed && statsEndWitnessed
}

func TestRunContainer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Log("building client")
	rawCl, err := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
	require.NoError(t, err)
	defer func() {
		if cErr := rawCl.Close(); cErr != nil {
			t.Logf("closing docker client: %s", cErr)
		}
	}()
	cl := docker.NewClient(rawCl)

	t.Log("pull test image")
	evs := make(chan docker.Event, 1024)
	pub := events.ChannelPublisher(evs)
	if err = cl.PullImage(ctx, docker.PullImage{Name: testImage}, pub); err != nil {
		t.Errorf("pulling image: %s", err.Error())
		return
	}
	close(evs)
	_, _, err = rawCl.ImageInspectWithRaw(ctx, testImage)
	require.NoError(t, err)

	t.Log("creating simple container")
	evs = make(chan docker.Event, 64)
	pub = events.ChannelPublisher(evs)
	dockerID, err := cl.CreateContainer(ctx, cproto.NewID(), cproto.RunSpec{
		ContainerConfig: container.Config{
			Image:      testImage,
			Entrypoint: []string{"cat", "/tmp/whatever"},
		},
		Archives: []cproto.RunArchive{
			{
				Path: "/tmp",
				Archive: []archive.Item{
					{
						Path:     "/whatever",
						Type:     tar.TypeReg,
						Content:  []byte("hello"),
						FileMode: 0o0777,
					},
				},
			},
		},
	}, pub)
	require.NoError(t, err)

	t.Log("running simple container")
	c, err := cl.RunContainer(ctx, ctx, dockerID, pub)
	require.NoError(t, err)

	close(evs)
	defer func() {
		if err := rawCl.ContainerRemove(
			ctx,
			dockerID,
			types.ContainerRemoveOptions{Force: true},
		); err != nil {
			t.Errorf("failed to cleanup container %s: %s", dockerID, err)
		}
	}()

	select {
	case err := <-c.ContainerWaiter.Errs:
		t.Errorf("failed to wait for container: %s", err.Error())
	case exit := <-c.ContainerWaiter.Waiter:
		require.Equal(t, int64(0), exit.StatusCode)
		return
	}
}

const testServiceImage = "nginx:latest"

func TestRunContainerWithService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	agentID := uuid.NewString()
	containerID := uuid.NewString()

	t.Log("building client")
	rawCl, err := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
	require.NoError(t, err)
	defer func() {
		if cErr := rawCl.Close(); cErr != nil {
			t.Logf("closing docker client: %s", cErr)
		}
	}()
	cl := docker.NewClient(rawCl)

	t.Log("pull test image")
	evs := make(chan docker.Event, 1024)
	pub := events.ChannelPublisher(evs)
	if err = cl.PullImage(ctx, docker.PullImage{Name: testServiceImage}, pub); err != nil {
		t.Errorf("pulling image: %s", err.Error())
		return
	}
	close(evs)
	_, _, err = rawCl.ImageInspectWithRaw(ctx, testImage)
	require.NoError(t, err)

	t.Log("creating simple container")
	evs = make(chan docker.Event, 64)
	pub = events.ChannelPublisher(evs)
	dockerID, err := cl.CreateContainer(ctx, cproto.NewID(), cproto.RunSpec{
		ContainerConfig: container.Config{
			Image: testServiceImage,
			Labels: map[string]string{
				docker.AgentLabel:       agentID,
				docker.ContainerIDLabel: containerID,
			},
		},
	}, pub)
	require.NoError(t, err)

	t.Log("running simple container")
	c, err := cl.RunContainer(ctx, ctx, dockerID, pub)
	require.NoError(t, err)

	close(evs)
	defer func() {
		if rErr := rawCl.ContainerRemove(
			ctx,
			c.ContainerInfo.ID,
			types.ContainerRemoveOptions{Force: true},
		); rErr != nil {
			t.Errorf("failed to cleanup container %s: %s", c.ContainerInfo.ID, rErr)
		}
	}()

	t.Log("ensure it is listed when searching docker for our containers")
	containers, err := cl.ListRunningContainers(ctx, docker.LabelFilter(docker.AgentLabel, agentID))
	require.NoError(t, err)
	found := false
	for id := range containers {
		if id == cproto.ID(containerID) {
			found = true
			break
		}
	}
	require.True(t, found, "did not find our container")

	t.Log("ensure it can be reattached")
	reattached, terminated, err := cl.ReattachContainer(ctx, cproto.ID(containerID))
	require.NoError(t, err)
	require.Nil(t, terminated)

	t.Log("original waiters should exit after being killed")
	select {
	case <-time.After(time.Second):
		if err := cl.SignalContainer(ctx, c.ContainerInfo.ID, syscall.SIGTERM); err != nil {
			t.Errorf("failed to signal container: %s", err)
			return
		}
	case err := <-c.ContainerWaiter.Errs:
		t.Errorf("failed to wait for container: %s", err)
		return
	case exit := <-c.ContainerWaiter.Waiter:
		require.Equal(t, int64(0), exit.StatusCode)
		break
	}

	t.Log("reattached waiters should also exit after being killed")
	select {
	case err := <-reattached.ContainerWaiter.Errs:
		t.Errorf("failed to wait for reattached container: %s", err)
		return
	case exit := <-reattached.ContainerWaiter.Waiter:
		require.Equal(t, int64(0), exit.StatusCode)
		break
	}
}

//go:build integration

package containers_test

import (
	"context"
	"fmt"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	dcontainer "github.com/docker/docker/api/types/container"
	dclient "github.com/docker/docker/client"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/internal/containers"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/agent/test/testutils"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

//nolint:maintidx // Come on, it is a test.
func TestManager(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.SetLevel(log.TraceLevel)

	opts := testutils.DefaultAgentConfig(3)
	mopts := testutils.ElasticMasterSetAgentConfig()

	t.Log("building client")
	rawCl, err := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
	require.NoError(t, err)
	defer func() {
		if cErr := rawCl.Close(); cErr != nil {
			t.Logf("closing docker client: %s", cErr)
		}
	}()
	cl := docker.NewClient(rawCl)

	t.Log("setting up event handler")
	evs := make(chan container.Event, 5024)
	f := events.ChannelPublisher(evs)

	t.Log("creating container manager")
	m, err := containers.New(opts, mopts, nil, cl, f)
	require.NoError(t, err)
	defer m.Close()

	t.Log("running a few successful test containers")
	expectedSuccesses := map[cproto.ID]bool{}
	countSuccesses := 12
	for i := 0; i < countSuccesses; i++ {
		cID := cproto.ID(uuid.NewString())
		err = m.StartContainer(ctx, aproto.StartContainer{
			Container: cproto.Container{
				ID:      cID,
				State:   cproto.Assigned,
				Devices: []device.Device{},
			},
			Spec: cproto.Spec{
				TaskType: string(model.TaskTypeCommand),
				RunSpec: cproto.RunSpec{
					ContainerConfig: dcontainer.Config{
						Image:      "python",
						Entrypoint: []string{"echo", "hello"},
					},
				},
			},
		})
		require.NoError(t, err)
		expectedSuccesses[cID] = true
	}

	t.Log("running a few unsuccessful test containers")
	countFailures := 12
	expectedFailures := map[cproto.ID]bool{}
	for i := 0; i < countFailures; i++ {
		cID := cproto.ID(uuid.NewString())
		err = m.StartContainer(ctx, aproto.StartContainer{
			Container: cproto.Container{
				ID:      cID,
				State:   cproto.Assigned,
				Devices: []device.Device{},
			},
			Spec: cproto.Spec{
				TaskType: string(model.TaskTypeCommand),
				RunSpec: cproto.RunSpec{
					ContainerConfig: dcontainer.Config{
						Image:      "python",
						Entrypoint: []string{"ghci"}, // Ha..
					},
				},
			},
		})
		require.NoError(t, err)
		expectedFailures[cID] = true
	}

	t.Log("running a few longrunning test containers")
	countLongrunning := 12
	expectedLongrunning := map[cproto.ID]bool{}
	for i := 0; i < countLongrunning; i++ {
		cID := cproto.ID(uuid.NewString())
		err = m.StartContainer(ctx, aproto.StartContainer{
			Container: cproto.Container{
				ID:      cID,
				State:   cproto.Assigned,
				Devices: []device.Device{},
			},
			Spec: cproto.Spec{
				TaskType: string(model.TaskTypeCommand),
				RunSpec: cproto.RunSpec{
					ContainerConfig: dcontainer.Config{
						Image:      "python",
						Entrypoint: []string{"sleep", "99"},
					},
				},
			},
		})
		require.NoError(t, err)
		expectedLongrunning[cID] = true
	}

	t.Log("watching for container state changed for non-longrunning")
	actualStops := map[cproto.ID]*aproto.ContainerStopped{}
	actualSuccesses := map[cproto.ID]bool{}
	actualFailures := map[cproto.ID]bool{}
	deadline := time.After(30 * time.Second)
	for {
		select {
		case ev := <-evs:
			if ev.StateChange == nil || ev.StateChange.ContainerStopped == nil {
				continue
			}
			sc := ev.StateChange

			actualStops[sc.Container.ID] = sc.ContainerStopped
			if sc.ContainerStopped.Failure != nil {
				actualFailures[sc.Container.ID] = true
			} else {
				actualSuccesses[sc.Container.ID] = true
			}

			if len(actualSuccesses) == len(expectedSuccesses) &&
				len(actualFailures) == len(expectedFailures) {
				goto DONE
			}
		case <-deadline:
			t.Logf("did not receive state changes for non-longrunning in time")
			t.Logf(
				"want %s and %s, got %s",
				spew.Sdump(expectedSuccesses),
				spew.Sdump(expectedFailures),
				spew.Sdump(actualStops),
			)
			t.Fail()
			return
		}
	}
DONE:

	t.Log("checking results")
	require.Equalf(t, len(expectedFailures), len(actualFailures),
		"want: %s\ngot: %s", spew.Sdump(expectedFailures), spew.Sdump(actualFailures))

	t.Logf("trying to signal %d longrunning containers", len(expectedLongrunning))
	for cID := range expectedLongrunning {
		m.SignalContainer(ctx, aproto.SignalContainer{
			ContainerID: cID,
			Signal:      syscall.SIGKILL,
		})
	}

	t.Log("watching for container state changed for longrunning")
	actualLongrunning := map[cproto.ID]bool{}
	for ev := range evs {
		if ev.StateChange == nil || ev.StateChange.ContainerStopped == nil {
			continue
		}
		ev := ev.StateChange

		f := ev.ContainerStopped.Failure
		require.NotNil(t, f)
		if f.ExitCode != nil {
			require.Equal(t, aproto.ExitCode(137), *f.ExitCode)
		} else {
			require.Contains(t, f.Error(), "killed")
		}

		actualStops[ev.Container.ID] = ev.ContainerStopped
		actualLongrunning[ev.Container.ID] = true
		if len(expectedLongrunning) == len(actualLongrunning) {
			break
		}
	}

	// This is needed because of a sort of unfortunate race:
	//  1. We get the container exited for a container, cool.
	//  2. We resend another signal to see if we get a resent exit, cool.
	//  3. Oh no! The manager hasn't actually realized the container exited. It just puts the signal
	//     on the queue.
	//  4. We wait forever for the resent exit, and it never comes because nothing is reading
	//     signals off the queue anymore.
	// Oh no... some sort of RAII thing is going on here - the container is closed, but we can
	// still take actions on it.
	for m.NumContainers() != 0 {
		time.Sleep(time.Second)
	}

	t.Logf("trying to signal all containers, again, to check cache bust")
	for cID := range actualSuccesses {
		m.SignalContainer(ctx, aproto.SignalContainer{
			ContainerID: cID,
			Signal:      syscall.SIGKILL,
		})
	}
	for cID := range actualFailures {
		m.SignalContainer(ctx, aproto.SignalContainer{
			ContainerID: cID,
			Signal:      syscall.SIGKILL,
		})
	}
	for cID := range actualLongrunning {
		m.SignalContainer(ctx, aproto.SignalContainer{
			ContainerID: cID,
			Signal:      syscall.SIGKILL,
		})
	}

	t.Log("getting cached resends for all, cache busted responses for some")
	resentStops := map[cproto.ID]*aproto.ContainerStopped{}
	for ev := range evs {
		if ev.StateChange == nil && ev.StateChange.ContainerStopped == nil {
			continue
		}
		ev := ev.StateChange

		require.Nil(t, resentStops[ev.Container.ID])
		resentStops[ev.Container.ID] = ev.ContainerStopped
		if len(actualSuccesses)+len(actualFailures)+len(actualLongrunning) == len(resentStops) {
			break
		}
	}

	t.Log("checking number of stops exceeded cache, but uncached stops were still sent")
	uncachedStops := 0
	for _, stop := range resentStops {
		if stop.Failure == container.ErrMissing {
			uncachedStops++
		}
	}
	totalStops := len(actualSuccesses) + len(actualFailures) + len(actualLongrunning)
	expectedUncachedStops := totalStops - containers.RecentExitsCacheSize
	require.Equal(t, expectedUncachedStops, uncachedStops, spew.Sdump(resentStops))
}

func TestManagerReattach(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logC := testutils.NewLogChannel(1024)
	log.SetLevel(log.TraceLevel)
	log.AddHook(logC)

	opts := testutils.DefaultAgentConfig(4)
	opts.AgentID = uuid.NewString()
	mopts := testutils.ElasticMasterSetAgentConfig()

	t.Log("building client")
	rawCl, err := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
	require.NoError(t, err)
	defer func() {
		if cErr := rawCl.Close(); cErr != nil {
			t.Logf("closing docker client: %s", cErr)
		}
	}()
	cl := docker.NewClient(rawCl)

	t.Log("setting up event handler")
	evs := make(chan container.Event, 5024)
	f := events.ChannelPublisher(evs)

	t.Log("creating container manager")
	m, err := containers.New(opts, mopts, nil, cl, f)
	require.NoError(t, err)
	defer m.Close()

	t.Log("running a few longrunning test containers")
	starts := map[cproto.ID]bool{}
	defer func() {
		t.Log("beginning deferred cleanup")
		conts, lErr := cl.ListRunningContainers(ctx, docker.LabelFilter(docker.AgentLabel, opts.AgentID))
		require.NoError(t, lErr)

		for cID, cont := range conts {
			if rErr := cl.RemoveContainer(ctx, cont.ID, true); rErr != nil {
				t.Logf("cleanup of %s failed with %s", cID, rErr.Error())
			}
		}
	}()

	countLongrunning := 12
	for i := 0; i < countLongrunning; i++ {
		cID := cproto.ID(uuid.NewString())
		err = m.StartContainer(ctx, aproto.StartContainer{
			Container: cproto.Container{
				ID:      cID,
				State:   cproto.Assigned,
				Devices: []device.Device{},
			},
			Spec: cproto.Spec{
				TaskType: string(model.TaskTypeCommand),
				RunSpec: cproto.RunSpec{
					ContainerConfig: dcontainer.Config{
						Image:      "python",
						Entrypoint: []string{"sleep", "5"},
					},
					HostConfig: dcontainer.HostConfig{AutoRemove: true},
				},
			},
		})
		require.NoError(t, err)
		starts[cID] = true
	}

	t.Log("waiting for containers to be running")
	var expectedSurvivors []aproto.ContainerReattach
	for ev := range evs {
		if ev.StateChange == nil {
			continue
		}

		ev := ev.StateChange
		switch ev.Container.State {
		case cproto.Terminated:
			require.Truef(t, false, "container exited unexpectedly %s", spew.Sdump(ev))
		case cproto.Running:
		default:
			continue
		}

		expectedSurvivors = append(expectedSurvivors, aproto.ContainerReattach{Container: ev.Container})
		if len(expectedSurvivors) == countLongrunning {
			break
		}
	}

	t.Log("remove one container from expected, to see if we cull it")
	unexpectedSurvivor, expectedSurvivors := expectedSurvivors[0], expectedSurvivors[1:]

	t.Log("killing one expected survivor, to see if we notice it is gone")
	missingSurvivor := expectedSurvivors[0]
	conts, err := cl.ListRunningContainers(ctx, docker.LabelFilter(
		docker.ContainerIDLabel,
		missingSurvivor.Container.ID.String(),
	))
	require.NoError(t, err)
	require.Len(t, conts, 1)
	err = cl.RemoveContainer(ctx, conts[missingSurvivor.Container.ID].ID, true)
	require.NoError(t, err)

	t.Log("detaching from all containers")
	m.Detach()

	t.Log("reattaching all containers")
	acks, err := m.ReattachContainers(ctx, expectedSurvivors)
	require.NoError(t, err)
	require.Equal(t, len(expectedSurvivors), len(acks))

	t.Log("checking the unexpected survivor is culled")
	found := false
	for l := range logC {
		if strings.Contains(
			l.Message,
			fmt.Sprintf("will kill container %s", unexpectedSurvivor.Container.ID),
		) {
			found = true
			break
		}
	}
	require.True(t, found, "no indication we culled unexpected container")

	t.Logf("waiting for %d reattached containers to exit, happily", len(expectedSurvivors))
	terminated := 0
	for ev := range evs {
		if ev.StateChange == nil || ev.StateChange.ContainerStopped == nil {
			continue
		}

		ev := ev.StateChange
		switch ev.Container.State {
		case cproto.Terminated:
			terminated++
			t.Logf("reattached %d/%d", terminated, len(expectedSurvivors))
			if ev.Container.ID == missingSurvivor.Container.ID {
				require.NotNil(t, ev.ContainerStopped)
				require.NotNil(t, ev.ContainerStopped.Failure)
			} else {
				require.NotNil(t, ev.ContainerStopped)
				require.Nil(t, ev.ContainerStopped.Failure)
			}
		default:
			continue
		}

		if len(expectedSurvivors) == terminated {
			return
		}
	}
}

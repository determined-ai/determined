//go:build integration

package containers_test

import (
	"context"
	"fmt"
	"strings"
	"syscall"
	"testing"

	"github.com/davecgh/go-spew/spew"
	dcontainer "github.com/docker/docker/api/types/container"
	dclient "github.com/docker/docker/client"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/internal/containers"
	"github.com/determined-ai/determined/agent/internal/fluent"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/agent/test/testutils"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestManager(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.SetLevel(log.TraceLevel)

	opts := testutils.DefaultAgentConfig(3)
	mopts := testutils.ElasticMasterSetAgentConfig()

	t.Log("building client")
	rawCl, err := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
	require.NoError(t, err)
	cl := docker.NewClient(rawCl)
	defer func() {
		if cErr := cl.Close(); cErr != nil {
			t.Logf("closing docker client: %s", cErr)
		}
	}()

	t.Log("starting fluent")
	fl, err := fluent.Start(ctx, opts, mopts, cl)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, fl.Close())
	}()

	t.Log("setting up event handler")
	evs := make(chan *aproto.ContainerStateChanged, 5024) // Buffer it a ton, as not don't pipeline.
	f := events.FuncPublisher[container.Event](func(ctx context.Context, e container.Event) error {
		sc := e.StateChange
		if sc == nil {
			return nil
		}

		cs := sc.ContainerStopped
		if cs == nil {
			return nil
		}
		evs <- sc
		return nil
	})

	t.Log("creating container manager")
	m, err := containers.New(opts, mopts, nil, cl, f)
	require.NoError(t, err)
	defer m.Close()

	t.Log("running a few successful test containers")
	starts := map[cproto.ID]bool{}
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
						Image:      "ubuntu",
						Entrypoint: []string{"echo", "hello"},
					},
				},
			},
		})
		require.NoError(t, err)
		starts[cID] = true
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
						Image:      "ubuntu",
						Entrypoint: []string{"ghci"}, // Ha..
					},
				},
			},
		})
		require.NoError(t, err)
		starts[cID] = true
		expectedFailures[cID] = true
	}

	t.Log("running a few longrunning test containers")
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
						Image:      "ubuntu",
						Entrypoint: []string{"sleep", "99"},
					},
				},
			},
		})
		require.NoError(t, err)
		starts[cID] = true
	}

	t.Log("watching for container state changed")
	stops := map[cproto.ID]*aproto.ContainerStopped{}
	failures := map[cproto.ID]bool{}
	for ev := range evs {
		stops[ev.Container.ID] = ev.ContainerStopped
		if ev.ContainerStopped.Failure != nil {
			failures[ev.Container.ID] = true
		}
		if countSuccesses+countFailures == len(stops) {
			break
		}
	}

	t.Log("checking results")
	require.Equal(t, countSuccesses+countFailures, len(stops))
	require.Equalf(t, countFailures, len(failures),
		"want: %s\ngot: %s", spew.Sdump(expectedFailures), spew.Sdump(failures))

	t.Logf("trying to signal %d the containers", len(starts))
	for cID := range starts {
		m.SignalContainer(ctx, aproto.SignalContainer{
			ContainerID: cID,
			Signal:      syscall.SIGKILL,
		})
	}

	t.Log("watching for container state changed for longrunning, cached resends for others")
	stops = map[cproto.ID]*aproto.ContainerStopped{}
	failures = map[cproto.ID]bool{}
	for ev := range evs {
		stops[ev.Container.ID] = ev.ContainerStopped
		if ev.ContainerStopped.Failure != nil {
			failures[ev.Container.ID] = true
		}
		if len(starts) == len(stops) {
			break
		}
	}

	t.Log("checking resend and longrunning results")
	require.Equal(t, len(starts), len(stops))
	require.Equal(t, countFailures+countLongrunning, len(failures))

	t.Logf("trying to signal %d the containers, again, to check cache bust", len(starts))
	for cID := range starts {
		m.SignalContainer(ctx, aproto.SignalContainer{
			ContainerID: cID,
			Signal:      syscall.SIGKILL,
		})
	}

	t.Log("getting cached resends for all, cache busted responses for some")
	stops = map[cproto.ID]*aproto.ContainerStopped{}
	failures = map[cproto.ID]bool{}
	for ev := range evs {
		stops[ev.Container.ID] = ev.ContainerStopped
		if ev.ContainerStopped.Failure != nil {
			failures[ev.Container.ID] = true
		}
		if len(starts) == len(stops) {
			break
		}
	}

	t.Log("checking number of stops exceeded cache, but uncached stops were still sent")
	uncachedStops := 0
	for _, stop := range stops {
		if strings.Contains(stop.String(), container.ErrMissing.ErrMsg) {
			uncachedStops++
		}
	}
	require.Equal(t, countSuccesses+countFailures+countLongrunning-32, uncachedStops)
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
	cl := docker.NewClient(rawCl)
	defer func() {
		if cErr := cl.Close(); cErr != nil {
			t.Logf("closing docker client: %s", cErr)
		}
	}()

	t.Log("starting fluent")
	fl, err := fluent.Start(ctx, opts, mopts, cl)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, fl.Close())
	}()

	t.Log("setting up event handler")
	evs := make(chan *aproto.ContainerStateChanged, 5024) // Buffer it a ton, as not don't pipeline.
	f := events.FuncPublisher[container.Event](func(ctx context.Context, e container.Event) error {
		sc := e.StateChange
		if sc == nil {
			return nil
		}

		evs <- sc
		return nil
	})

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
						Image:      "ubuntu",
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
		switch ev.Container.State {
		case cproto.Terminated:
			require.Truef(t, false, "container exited unexpectedly %s", spew.Sdump(ev))
		case cproto.Running:
			break
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

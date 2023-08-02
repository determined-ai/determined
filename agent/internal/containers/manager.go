package containers

import (
	"container/ring"
	"context"
	"fmt"
	"sync"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	"golang.org/x/sys/unix"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

const (
	httpInsecureScheme = "http"
	httpSecureScheme   = "https"
	// RecentExitsCacheSize is the number of cached stops we keep, before forgetting about them.
	RecentExitsCacheSize = 32
)

// Manager manages containers. It is able to start and signal them and tracks some updates to their
// state.
type Manager struct {
	// Configuration details. Set in initialization and never modified after.
	opts    options.Options
	mopts   aproto.MasterSetAgentOptions
	devices []device.Device

	// System dependencies. Also set in initialization and never modified after.
	log      *log.Entry
	cruntime container.ContainerRuntime
	pub      events.Publisher[container.Event]

	// Internal state. Access should be protected.
	containers  map[cproto.ID]*container.Container
	recentExits *ring.Ring
	wg          waitgroupx.Group
	mu          sync.RWMutex
}

// New returns a new container manager.
func New(
	opts options.Options,
	mopts aproto.MasterSetAgentOptions,
	devices []device.Device,
	cl container.ContainerRuntime,
	pub events.Publisher[container.Event],
) (*Manager, error) {
	return &Manager{
		opts:        opts,
		mopts:       mopts,
		devices:     devices,
		log:         log.WithField("component", "container-manager"),
		cruntime:    cl,
		pub:         pub,
		containers:  make(map[cproto.ID]*container.Container),
		recentExits: ring.New(RecentExitsCacheSize),
		wg:          waitgroupx.WithContext(context.Background()), // Manager-scoped group.
	}, nil
}

// ReattachContainers takes a list of expected survivors and returns the results of the attempted
// reattach. A result is returned for every expected survivor. An error indicates a total failure.
func (m *Manager) ReattachContainers(
	ctx context.Context, expectedSurvivors []aproto.ContainerReattach,
) ([]aproto.ContainerReattachAck, error) {
	m.log.Debugf("reattachContainers: expected survivors: %v", expectedSurvivors)
	result := make([]aproto.ContainerReattachAck, 0, len(expectedSurvivors))

	agentFilter := docker.LabelFilter(docker.AgentLabel, m.opts.AgentID)
	runningContainers, err := m.cruntime.ListRunningContainers(ctx, agentFilter)
	if err != nil {
		return nil, err
	}
	m.log.Debugf("reattachContainers: running containers: %v", maps.Keys(runningContainers))

	m.log.Trace("iterating expected survivors and seeing if they were found")
	for _, expectedSurvivor := range expectedSurvivors {
		cID := expectedSurvivor.Container.ID

		var ack aproto.ContainerReattachAck

		containerInfo, ok := runningContainers[cID]
		if !ok {
			m.log.Tracef("container is gone on reattachment %s", cID)
			ack = aproto.ContainerReattachAck{
				Container: cproto.Container{ID: cID},
				Failure: &aproto.ContainerFailure{
					FailureType: aproto.RestoreError,
					ErrMsg:      "container is gone on reattachment",
				},
			}
		} else {
			m.log.Infof("will reattach container %s", cID)
			cpc, err := m.reattachContainer(ctx, expectedSurvivor.Container, containerInfo)
			if err != nil {
				err = fmt.Errorf("failed to restore info from container labels: %w", err)
				m.log.WithError(err).Tracef("failed to reattach container %s", cID)
				ack = aproto.ContainerReattachAck{
					Container: cproto.Container{ID: cID},
					Failure: &aproto.ContainerFailure{
						FailureType: aproto.RestoreError,
						ErrMsg:      err.Error(),
					},
				}
			} else {
				m.log.Tracef("successfully reattached container %s", cID)
				ack = aproto.ContainerReattachAck{
					Container: *cpc,
				}
			}
		}

		result = append(result, ack)
		delete(runningContainers, cID)
	}

	m.log.Trace("sending SIGKILL to running containers that were not reattached")
	for cid, containerInfo := range runningContainers {
		m.log.Infof("will kill container %s", cid)
		if err := m.cruntime.SignalContainer(ctx, containerInfo.ID, unix.SIGKILL); err != nil {
			m.log.WithError(err).Warnf("failed to kill container %s", cid)
		}
	}

	return result, nil
}

// RevalidateContainers rectifies a list of containers the mananger is expected to know about with
// what the manager does know about, and returns updates about the expected containers.
func (m *Manager) RevalidateContainers(
	ctx context.Context, expectedSurvivors []aproto.ContainerReattach,
) ([]aproto.ContainerReattachAck, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]aproto.ContainerReattachAck, 0, len(expectedSurvivors))
	for _, expectedSurvivor := range expectedSurvivors {
		cid := expectedSurvivor.Container.ID

		// If the child is still there, assuming nothing has changed.
		c, ok := m.containers[cid]
		if ok {
			result = append(result, aproto.ContainerReattachAck{Container: c.Summary()})
			continue
		}

		// If there is a termination message for it, for any reason, go ahead and ack that.
		var ack *aproto.ContainerReattachAck
		m.recentExits.Do(func(v any) {
			if v == nil {
				return
			}

			savedStop := v.(*aproto.ContainerStateChanged)
			if cid != savedStop.Container.ID {
				return
			}

			ack = &aproto.ContainerReattachAck{
				Container: savedStop.Container,
				Failure:   savedStop.ContainerStopped.Failure,
			}
		})
		if ack != nil {
			result = append(result, *ack)
			continue
		}

		// Else fallback to a missing message.
		result = append(result, aproto.ContainerReattachAck{
			Container: cproto.Container{ID: cid},
			Failure: &aproto.ContainerFailure{
				FailureType: aproto.RestoreError,
				ErrMsg:      "failed to restore container on master blip",
			},
		})
	}
	return result, nil
}

// StartContainer starts a container according to the provided spec, relaying its state changes via
// events.
func (m *Manager) StartContainer(ctx context.Context, req aproto.StartContainer) error {
	m.log.Tracef("starting container %s", req.Container.ID)
	if !validateDevices(m.devices, req.Container.Devices) {
		return fmt.Errorf("devices specified in container spec not found on agent")
	}

	spec, err := overwriteSpec(req.Spec, req.Container, m.opts, m.mopts)
	if err != nil {
		return fmt.Errorf("failed to overwrite spec: %w", err)
	}
	req.Spec = spec

	m.mu.Lock()
	if m.containers[req.Container.ID] != nil {
		m.mu.Unlock()
		return fmt.Errorf("container already created: %s", req.Container.ID)
	}
	c := container.Start(req, m.cruntime, m.pub)
	m.containers[req.Container.ID] = c
	m.mu.Unlock()

	m.wg.Go(func(_ context.Context) {
		exit := c.Wait()
		m.mu.Lock()
		if exit != nil {
			m.recentExits = m.recentExits.Prev()
			m.recentExits.Value = exit
		}
		delete(m.containers, req.Container.ID)
		m.mu.Unlock()
	})
	return nil
}

// SignalContainer signals a container.
func (m *Manager) SignalContainer(ctx context.Context, msg aproto.SignalContainer) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cont, ok := m.containers[msg.ContainerID]
	if !ok {
		exit := m.recentExit(msg.ContainerID, container.ErrMissing)
		m.log.Warnf("resending stop for missing container: %v", exit)
		if err := m.pub.Publish(ctx, container.Event{StateChange: exit}); err != nil {
			m.log.WithError(err).Errorf("failed to resend stop")
		}
		return
	}

	cont.Signal(ctx, msg.Signal)
}

// Detach from all running containers without affecting them.
func (m *Manager) Detach() {
	m.mu.RLock()
	for _, c := range m.containers {
		c := c
		m.wg.Go(func(_ context.Context) {
			c.Detach()
		})
	}
	m.mu.RUnlock()
	m.wg.Wait()
}

// Close all managed containers by sending them a SIGKILL and wait for them to close.
func (m *Manager) Close() {
	m.mu.RLock()
	for _, c := range m.containers {
		c := c
		m.wg.Go(func(_ context.Context) {
			c.Stop()
		})
	}
	m.mu.RUnlock()
	m.wg.Wait()
}

// NumContainers returns the number of containers being managed.
func (m *Manager) NumContainers() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.containers)
}

func (m *Manager) reattachContainer(
	ctx context.Context, containerPrevState cproto.Container, containerInfo types.Container,
) (*cproto.Container, error) {
	cID := containerPrevState.ID

	containerCurrState, err := m.unmakeContainerDockerLabels(containerInfo)
	if err != nil {
		return nil, err
	}
	// TODO(ilia): Support reattaching containers that have changed state:
	// - starting -> running,
	// - running -> terminated.
	if containerPrevState.State != "" && containerCurrState.State != containerPrevState.State {
		return nil, fmt.Errorf(
			"container has changed state while offline. now: %s, was: %s",
			containerCurrState.State, containerPrevState.State,
		)
	}

	m.mu.Lock()
	if c, ok := m.containers[cID]; ok {
		m.mu.Unlock()
		errorMsg := fmt.Sprintf("failed to reattach container %s: handle already exists", cID)
		m.log.Warnf(errorMsg)
		m.log.Warnf("possible invalid state, killed container actor %s", cID)
		c.Signal(ctx, syscall.SIGKILL)
		return nil, errors.New(errorMsg)
	}
	c := container.Reattach(*containerCurrState, m.cruntime, m.pub)
	m.containers[cID] = c
	m.mu.Unlock()

	m.wg.Go(func(_ context.Context) {
		exit := c.Wait()
		m.mu.Lock()
		if exit != nil {
			m.recentExits = m.recentExits.Prev()
			m.recentExits.Value = exit
		}
		delete(m.containers, cID)
		m.mu.Unlock()
	})
	m.log.Debugf("reattached container actor %s", cID)
	return containerCurrState, nil
}

func (m *Manager) recentExit(
	cID cproto.ID,
	fallback *aproto.ContainerFailure,
) *aproto.ContainerStateChanged {
	var stop *aproto.ContainerStateChanged
	m.recentExits.Do(func(v any) {
		if v == nil {
			return
		}

		savedStop := v.(*aproto.ContainerStateChanged)
		if cID != savedStop.Container.ID {
			return
		}
		stop = savedStop
	})

	if stop != nil {
		return stop
	}
	return &aproto.ContainerStateChanged{
		Container: cproto.Container{
			ID:    cID,
			State: cproto.Terminated,
		},
		ContainerStopped: &aproto.ContainerStopped{
			Failure: fallback,
		},
	}
}

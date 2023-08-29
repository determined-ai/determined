package container

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

// Container is a layer for managing a single Docker container. It can be constructed by launching
// a new container or reattaching an existing one. Once constructed, it provides an interface to
// interact with a running container.
type Container struct {
	// Configuration details. Set in initialization and never modified after.
	containerID  cproto.ID
	allocationID model.AllocationID
	spec         *cproto.Spec
	devices      []device.Device

	// System dependencies. Also set in initialization and never modified after.
	log      *logrus.Entry
	cruntime ContainerRuntime
	pub      events.Publisher[Event]

	// Internal state. Access should be protected.
	mu       sync.RWMutex
	state    cproto.State // Updated throughout run, access protected.
	signals  chan syscall.Signal
	exit     *aproto.ContainerStateChanged // Always set if the container exits.
	exitOnce sync.Once

	wg   waitgroupx.Group // A container-scoped goroutine group.
	done chan struct{}    // Closed after the group terminates and we finalize our state.
}

// Start a container asynchronously and receive a handle to interact with it.
func Start(
	req aproto.StartContainer,
	cl ContainerRuntime,
	pub events.Publisher[Event],
) *Container {
	c := &Container{
		containerID:  req.Container.ID,
		allocationID: hackAllocationID(&req.Spec),
		spec:         &req.Spec,
		devices:      req.Container.Devices,
		log: logrus.WithFields(logrus.Fields{
			"component": "container",
			"cproto-id": req.Container.ID,
		}),
		cruntime: cl,
		pub:      pub,
		state:    req.Container.State,
		signals:  make(chan syscall.Signal),
		done:     make(chan struct{}),

		wg: waitgroupx.WithContext(context.Background()),
	}

	c.wg.Go(func(ctx context.Context) {
		defer c.wg.Cancel()
		c.finalize(ctx, c.run(ctx))
	})

	go func() {
		c.wg.Wait()
		close(c.done)
	}()

	return c
}

// Reattach an existing container and receive a handle to interact with it.
func Reattach(
	container cproto.Container,
	cl ContainerRuntime,
	pub events.Publisher[Event],
) *Container {
	c := &Container{
		// TODO(Brad): We should be recovering the allocation ID for logging.
		containerID: container.ID,
		// We don't need the spec because we only reattach launched containers.
		devices: container.Devices,
		log: logrus.WithFields(logrus.Fields{
			"component":  "container",
			"cproto-id":  container.ID,
			"reattached": true,
		}),
		cruntime: cl,
		pub:      pub,
		state:    container.State,
		signals:  make(chan syscall.Signal, 16), // Not infinite, but large enough to not drop often.
		done:     make(chan struct{}),

		wg: waitgroupx.WithContext(context.Background()),
	}

	c.wg.Go(func(ctx context.Context) {
		defer c.wg.Cancel()
		c.finalize(ctx, c.reattach(ctx))
	})

	go func() {
		c.wg.Wait()
		close(c.done)
	}()

	return c
}

// Summary returns a snapshot of the container state.
func (c *Container) Summary() cproto.Container {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.summary()
}

// Detach the monitoring loops without affecting the Docker container.
func (c *Container) Detach() {
	c.log.Trace("detach called")
	c.wg.Cancel()
	c.Wait()
}

// Stop the Docker container by killing it and awaiting its exit.
func (c *Container) Stop() {
	c.log.Trace("close called")
	c.Signal(context.TODO(), syscall.SIGKILL)
	c.Wait()
}

// Signal asynchronously delivers the signal. Delivery failures are surfaced in logs.
func (c *Container) Signal(ctx context.Context, s syscall.Signal) {
	select {
	case c.signals <- s:
	case <-c.done:
		c.log.Warnf("ignoring signal on exited container: %v", s)
	case <-ctx.Done():
		c.log.Warnf("ignoring signal on container due to cancellation: %v", s)
	}
}

// Wait until the container exits. Always returns a ContainerExit unless canceled by Detach.
func (c *Container) Wait() *aproto.ContainerStateChanged {
	<-c.done
	return c.exit
}

// run the container. If the context is canceled, the container is detached (as is) and the context
// error is returned. If the container is killed, the container is cleaned up and a termination with
// a stack trace or kill result is returned. If the container succeeds, the error will be nil.
func (c *Container) run(parent context.Context) (err error) {
	c.log.Trace("starting container launch")
	launchgroup := errgroupx.WithContext(parent)

	c.log.Trace("kicking off goroutine shim SIGKILL to cancellations, until we have launched")
	launchgroup.Go(func(ctx context.Context) error {
		for {
			select {
			case signal := <-c.signals:
				switch signal {
				case syscall.SIGKILL:
					c.log.Tracef("signal %s, canceling run-scoped context", signal)
					return ErrKilledBeforeRun
				default:
					c.log.Warnf("ignoring signal other than SIGKILL %s before running", signal)
				}
			case <-ctx.Done():
				c.log.Trace("signal-to-context shimmer exited")
				return nil
			}
		}
	})

	c.log.Trace("kicking off goroutine to launch the container")
	var dockerContainer *docker.Container
	launchgroup.Go(func(ctx context.Context) (err error) {
		defer launchgroup.Cancel()

		c.log.Trace("pulling image")
		if err = c.transition(ctx, cproto.Pulling, nil, nil); err != nil {
			return err
		}
		if err = c.cruntime.PullImage(ctx, docker.PullImage{
			Name:      c.spec.RunSpec.ContainerConfig.Image,
			Registry:  c.spec.PullSpec.Registry,
			ForcePull: c.spec.PullSpec.ForcePull,
		}, c.shimDockerEvents()); err != nil {
			return fmt.Errorf("pulling container image: %w", err)
		}

		c.log.Trace("creating container, copying files, etc")
		if err = c.transition(ctx, cproto.Starting, nil, nil); err != nil {
			return err
		}

		runtimeID, err := c.cruntime.CreateContainer(
			ctx,
			c.containerID,
			c.spec.RunSpec,
			c.shimDockerEvents(),
		)
		if err != nil {
			return fmt.Errorf("creating container: %w", err)
		}
		remove := c.spec.RunSpec.HostConfig.AutoRemove
		c.spec = nil // Evict the spec from memory due to their potential memory consumption.
		defer func() {
			if err != nil {
				c.log.Trace("ensuring cleanup of container (canceled prior to the monitoring loop)")
				if remove {
					if rErr := c.cruntime.RemoveContainer(parent, runtimeID, true); rErr != nil {
						c.log.WithError(rErr).Debug("couldn't cleanup container")
					}
				}
				return
			}
		}()

		c.log.WithField("docker-id", runtimeID).Trace("starting container")
		dc, err := c.cruntime.RunContainer(ctx, parent, runtimeID, c.shimDockerEvents())
		if err != nil {
			return fmt.Errorf("starting container: %w", err)
		}
		dockerContainer = dc
		return nil
	})

	c.log.Trace("waiting for launch to complete")
	switch err := launchgroup.Wait(); {
	case err != nil && dockerContainer != nil:
		// There is a chance the launchgroup handled a signal, but that it happened after we
		// successfully ran the container. In this case, just pretend we didn't handle the signal,
		// give it back to the container, and continue.
		c.log.Trace("requeuing signal that was shimmed but unacknowledged")
		c.wg.Go(func(ctx context.Context) {
			select {
			case c.signals <- syscall.SIGKILL:
			case <-ctx.Done():
				c.log.Warnf("unable to re-enqueue signal due to cancellation")
			}
		})
	case err != nil:
		return err
	}

	c.log.Trace("transitioning to running state")
	if err := c.running(parent, aproto.ContainerStarted{
		ContainerInfo: dockerContainer.ContainerInfo,
	}); err != nil {
		return err
	}

	return c.wait(parent, dockerContainer)
}

func (c *Container) reattach(ctx context.Context) error {
	c.log.Trace("entering reattach")
	switch dc, exitCode, err := c.cruntime.ReattachContainer(
		ctx,
		c.containerID,
	); {
	case errors.Is(err, context.Canceled):
		return err
	case err != nil:
		return aproto.NewContainerFailure(aproto.RestoreError, err)
	case exitCode != nil:
		return aproto.NewContainerExit(*exitCode)
	case dc == nil:
		return ErrMissing
	default:
		return c.wait(ctx, dc)
	}
}

func (c *Container) wait(ctx context.Context, dc *docker.Container) error {
	c.log.Trace("in monitoring loop")
	for {
		select {
		case exit := <-dc.ContainerWaiter.Waiter:
			c.log.Tracef("container exited with %d", exit.StatusCode)
			if exit.Error != nil {
				return fmt.Errorf("receiving container exit: %s", exit.Error.Message)
			}
			return aproto.NewContainerExit(aproto.ExitCode(exit.StatusCode))

		case err := <-dc.ContainerWaiter.Errs:
			c.log.Trace("container waiter failed")
			return fmt.Errorf("failed while waiting for container to exit: %w", err)

		case signal := <-c.signals:
			c.log.Tracef("container signaled: %s", signal)
			if err := c.cruntime.SignalContainer(ctx, dc.ContainerInfo.ID, signal); err != nil {
				c.log.WithError(err).Errorf(
					"failed to signal %v with %v", dc.ContainerInfo.ID, signal,
				)
				return aproto.NewContainerFailure(aproto.ContainerFailed, err)
			}

		case <-ctx.Done():
			c.log.Trace("container context canceled")
			return ctx.Err()
		}
	}
}

func (c *Container) finalize(ctx context.Context, err error) {
	c.log.Trace("finalizing container exit")
	if ctx.Err() != nil {
		// There is a chance that cancellation and some other error raced, meaning we have a
		// valid error and a canceled context. In this case, we just go ahead with the detach
		// flow - on reattach callers can just reinspect the container.
		c.log.
			WithError(err).
			WithField("ctx-err", ctx.Err()).
			Warnf("orphaning container")
		return
	}

	var stop aproto.ContainerStopped
	switch err := err.(type) {
	case nil:
		stop = aproto.ContainerStopped{Failure: nil}
	case *aproto.ContainerFailure:
		stop = aproto.ContainerStopped{Failure: err}
	default:
		stop = aproto.ContainerError(aproto.TaskError, err)
	}

	if err := c.terminated(ctx, stop); err != nil {
		c.log.WithError(err).Error("finalizing container")
	}
	return
}

func (c *Container) summary() cproto.Container {
	return cproto.Container{
		ID:      c.containerID,
		State:   c.state,
		Devices: slices.Clone(c.devices),
	}
}

func (c *Container) transition(
	ctx context.Context,
	state cproto.State,
	start *aproto.ContainerStarted,
	stop *aproto.ContainerStopped,
) error {
	c.mu.Lock()
	c.log.WithField("stop", stop).Infof("transitioning state from %s to %s", c.state, state)
	c.state = state
	csc := &aproto.ContainerStateChanged{
		Container:        c.summary(),
		ContainerStarted: start,
		ContainerStopped: stop,
	}
	if c.state == cproto.Terminated {
		c.exitOnce.Do(func() { c.exit = csc })
	}
	c.mu.Unlock()

	if err := c.pub.Publish(ctx, Event{StateChange: csc}); err != nil {
		return fmt.Errorf("publishing %s event: %w", state, err)
	}
	return nil
}

func (c *Container) running(ctx context.Context, start aproto.ContainerStarted) error {
	return c.transition(ctx, cproto.Running, &start, nil)
}

func (c *Container) terminated(ctx context.Context, stop aproto.ContainerStopped) error {
	return c.transition(ctx, cproto.Terminated, nil, &stop)
}

func (c *Container) shimDockerEvents() events.Publisher[docker.Event] {
	return events.FuncPublisher[docker.Event](func(ctx context.Context, e docker.Event) error {
		switch {
		case e.Log != nil:
			source := "agent" // enrich log
			return c.pub.Publish(ctx, Event{Log: &aproto.ContainerLog{
				ContainerID: c.containerID,
				Timestamp:   e.Log.Timestamp,
				Level:       &e.Log.Level,
				AuxMessage:  &e.Log.Message,
				Source:      &source,
			}})

		case e.Stats != nil:
			var endStats bool
			switch {
			case e.Stats.StartTime != nil:
				endStats = false
			case e.Stats.EndTime != nil:
				endStats = true
			}

			return c.pub.Publish(ctx, Event{StatsRecord: &aproto.ContainerStatsRecord{
				EndStats: endStats,
				TaskType: model.TaskType(c.spec.TaskType),
				Stats: &model.TaskStats{
					AllocationID: c.allocationID,
					EventType:    e.Stats.Kind,
					StartTime:    e.Stats.StartTime,
					EndTime:      e.Stats.EndTime,
				},
			}})

		default:
			panic(fmt.Sprintf("unsupported docker event: %+v", e))
		}
	})
}

// hackAllocationID hacks the allocation ID back from the container spec.
// TODO(Brad): we should just.. pass this down?
func hackAllocationID(spec *cproto.Spec) model.AllocationID {
	for _, env := range spec.RunSpec.ContainerConfig.Env {
		split := strings.SplitN(env, "=", 2)
		if len(split) < 2 {
			continue
		}

		value := split[1]
		switch split[0] {
		case AllocationIDEnvVar:
			return model.AllocationID(value)
		}
	}
	return ""
}

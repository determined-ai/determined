//go:build integration

package container_test

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	dcontainer "github.com/docker/docker/api/types/container"
	dclient "github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

func TestContainer(t *testing.T) {
	log.SetLevel(log.TraceLevel)

	t.Log("building client")
	rawCl, err := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
	require.NoError(t, err)
	defer func() {
		if cErr := rawCl.Close(); cErr != nil {
			t.Logf("closing docker client: %s", cErr)
		}
	}()
	cl := docker.NewClient(rawCl)

	tests := []struct {
		name string

		image      string
		entrypoint []string

		detachAtState cproto.State
		signalAtState cproto.State
		signal        syscall.Signal

		failure *aproto.ContainerFailure
	}{
		{
			name:       "successful command",
			image:      "ubuntu",
			entrypoint: []string{"echo", "hello"},
			failure:    nil,
		},
		{
			name:       "non-existent image",
			image:      "lieblos/notanimageipushed",
			entrypoint: []string{"echo", "hello"},
			failure: &aproto.ContainerFailure{
				FailureType: aproto.TaskError,
				ErrMsg:      "repository does not exist or may require 'docker login'",
			},
		},
		{
			name:       "non-existent command",
			image:      "ubuntu",
			entrypoint: []string{"badcommandthatdoesntexit"},
			failure: &aproto.ContainerFailure{
				FailureType: aproto.TaskError,
				ErrMsg:      "executable file not found in $PATH",
			},
		},
		{
			name:       "failed command",
			image:      "ubuntu",
			entrypoint: []string{"ls", "badfile"},
			failure: &aproto.ContainerFailure{
				FailureType: aproto.ContainerFailed,
				ErrMsg:      "container failed with non-zero exit code",
				ExitCode:    (*aproto.ExitCode)(ptrs.Ptr(2)),
			},
		},
		{
			name:          "canceled during pull",
			image:         "pytorch/pytorch",
			entrypoint:    []string{"echo", "hello"},
			detachAtState: cproto.Pulling,
			failure: &aproto.ContainerFailure{
				FailureType: aproto.ContainerMissing,
				ErrMsg:      "container is gone on reattachment",
			},
		},
		{
			name:          "canceled during run, reattaches and exits ok",
			image:         "ubuntu",
			entrypoint:    []string{"sleep", "1"},
			detachAtState: cproto.Running,
		},
		{
			name:          "killed during pull",
			image:         "pytorch/pytorch",
			entrypoint:    []string{"echo", "hello"},
			signalAtState: cproto.Pulling,
			signal:        syscall.SIGKILL,
			failure: &aproto.ContainerFailure{
				FailureType: aproto.ContainerAborted,
				ErrMsg:      "killed before run",
			},
		},
		{
			name:          "killed during run",
			image:         "ubuntu",
			entrypoint:    []string{"sleep", "60"},
			signalAtState: cproto.Running,
			signal:        syscall.SIGKILL,
			failure: &aproto.ContainerFailure{
				FailureType: aproto.ContainerFailed,
				ErrMsg:      "137",
				ExitCode:    (*aproto.ExitCode)(ptrs.Ptr(137)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			t.Log("creating container")
			id := cproto.NewID()
			c := container.Start(aproto.StartContainer{
				Container: cproto.Container{
					ID:      id,
					State:   cproto.Assigned,
					Devices: []device.Device{},
				},
				Spec: cproto.Spec{
					TaskType: string(model.TaskTypeCommand),
					RunSpec: cproto.RunSpec{
						ContainerConfig: dcontainer.Config{
							Image:      tt.image,
							Entrypoint: tt.entrypoint,
							Labels:     map[string]string{docker.ContainerIDLabel: id.String()},
							Env:        []string{"DET_EXISTS", "DET_ALLOCATION_ID=3"},
						},
						HostConfig: dcontainer.HostConfig{AutoRemove: true},
					},
				},
			}, cl, events.NilPublisher[container.Event]{})
			defer c.Stop()

			t.Log("setup canceler")
			subg := waitgroupx.WithContext(ctx)
			subg.Go(func(ctx context.Context) {
				defer subg.Cancel()

				tck := time.NewTicker(10 * time.Millisecond)
				defer tck.Stop()
				for {
					switch summary := c.Summary(); {
					case summary.State == tt.detachAtState:
						t.Log("detaching container")
						c.Detach()
						return
					case summary.State == tt.signalAtState:
						t.Logf("signaling container: %s", tt.signal.String())
						c.Signal(ctx, tt.signal)
						return
					}

					select {
					case <-tck.C:
					case <-ctx.Done():
						return
					}
				}
			})
			defer subg.Wait()
			defer subg.Cancel()

			t.Log("waiting on container")
			exit := c.Wait()
			if tt.detachAtState != "" {
				t.Log("validating detach")
				require.Nilf(t, exit, "container exited but should've detached: %s", spew.Sdump(exit))

				t.Log("join canceler")
				subg.Cancel()
				subg.Wait()

				t.Log("reattaching container")
				c = container.Reattach(c.Summary(), cl, events.NilPublisher[container.Event]{})
				exit = c.Wait()
			}

			t.Log("interpreting container exit")
			require.NotNil(t, exit, "container returned without exiting")
			require.NotNil(t, exit.ContainerStopped, "container exit did not contain a stop")
			require.Equal(t, cproto.Terminated, exit.Container.State)

			failure := exit.ContainerStopped.Failure
			if tt.failure == nil {
				require.Nil(t, failure)
				return
			}

			require.NotNil(t, failure)
			require.Equal(t, tt.failure.FailureType, failure.FailureType, failure.Error())
			require.Equal(t, tt.failure.ExitCode, failure.ExitCode)
			require.Contains(t, failure.ErrMsg, tt.failure.ErrMsg)
		})
	}
}

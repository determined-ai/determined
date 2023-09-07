//go:build integration && singularity

package singularity_test

import (
	"context"
	"fmt"
	"os/user"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/agent/pkg/singularity"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// TODO(DET-9077): Get coverage to 70-80%.
func TestSingularity(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Log("creating client")
	cl, err := singularity.New(options.SingularityOptions{})
	require.NoError(t, err)

	t.Log("pulling container image")
	image := fmt.Sprintf("docker://%s", expconf.CPUImage)
	cprotoID := cproto.NewID()
	evs := make(chan docker.Event, 1024)
	pub := events.ChannelPublisher(evs)
	err = cl.PullImage(ctx, docker.PullImage{
		Name:     image,
		Registry: &types.AuthConfig{},
	}, pub)
	require.NoError(t, err)

	t.Log("creating container")
	u, err := user.Current()
	require.NoError(t, err)

	id, err := cl.CreateContainer(
		ctx,
		cprotoID,
		cproto.RunSpec{
			ContainerConfig: container.Config{
				Image: image,
				Cmd:   strslice.StrSlice{"/run/determined/train/entrypoint.sh"},
				Env:   []string{},
				User:  fmt.Sprintf("%s:%s", u.Uid, u.Gid),
			},
			HostConfig: container.HostConfig{
				NetworkMode: "host",
			},
			NetworkingConfig: network.NetworkingConfig{},
			Archives:         []cproto.RunArchive{},
		},
		pub,
	)
	require.NoError(t, err)

	t.Log("running container")
	waiter, err := cl.RunContainer(ctx, ctx, id, pub)
	require.NoError(t, err)

	select {
	case res := <-waiter.ContainerWaiter.Waiter:
		require.Nil(t, res.Error)
	case <-ctx.Done():
	}
}

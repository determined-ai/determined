package internal

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/cproto"
)

type tick struct{}

type checkerActor struct {
	client http.Client
	urls   []string
	period float64
}

func newCheckerActor(
	config cproto.ChecksConfig,
	info types.ContainerJSON,
) (*checkerActor, error) {
	urls := make([]string, 0, len(config.Checks))
	for _, config := range config.Checks {
		address, err := mapPortToHost(config.Port, info.NetworkSettings)
		if err != nil {
			return nil, err
		}
		urls = append(urls, fmt.Sprintf("http://%s/%s", address, config.Path))
	}
	return &checkerActor{
		client: http.Client{
			Timeout: time.Second,
		},
		urls:   urls,
		period: config.PeriodSeconds,
	}, nil
}

func (c *checkerActor) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart:
		c.scheduleCheck(ctx)
	case tick:
		for _, url := range c.urls {
			if !c.runCheck(ctx, url) {
				c.scheduleCheck(ctx)
				return nil
			}
		}

		ctx.Tell(ctx.Self().Parent(), containerReady{})
		ctx.Self().Stop()
	}
	return nil
}

func (c *checkerActor) scheduleCheck(ctx *actor.Context) {
	actors.NotifyAfter(ctx, time.Duration(c.period*float64(time.Second)), tick{})
}

func (c *checkerActor) runCheck(ctx *actor.Context, url string) bool {
	resp, err := c.client.Get(url)
	if err != nil {
		return false
	}
	if err := resp.Body.Close(); err != nil {
		return false
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return false
	}
	return true
}

// mapPortToHost takes a port number from a container's perspective and the network configuration of
// that port; from those, it determines an address and port connecting to which will effectively get
// us to that container on that port.
func mapPortToHost(port int, net *types.NetworkSettings) (string, error) {
	if len(net.Networks) == 0 {
		return "", errors.New("can't find any ports for a container with no networks")
	}

	// We require the agent to be using host networking (or running outside of Docker); thus, if the
	// container is using host networking, we simply use the given port number. Otherwise, we have to
	// examine the port mappings to find the corresponding port on the host.
	if _, ok := net.Networks["host"]; !ok {
		natPort, err := nat.NewPort("tcp", strconv.Itoa(port))
		if err != nil {
			return "", err
		}

		bindings := net.Ports[natPort]
		if len(bindings) == 0 {
			return "", errors.Errorf("could not find mapping for port %d", port)
		}

		// If there are multiple bindings, picking any one of them should be fine.
		port, err = strconv.Atoi(bindings[0].HostPort)
		if err != nil {
			return "", errors.Errorf("invalid port value %s", bindings[0].HostPort)
		}
	}
	return fmt.Sprintf("localhost:%d", port), nil
}

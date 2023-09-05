package internal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	dclient "github.com/docker/docker/client"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/internal/containers"
	"github.com/determined-ai/determined/agent/internal/detect"
	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/agent/pkg/podman"
	"github.com/determined-ai/determined/agent/pkg/singularity"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
	"github.com/determined-ai/determined/master/pkg/ws"
)

const (
	wsInsecureScheme = "ws"
	wsSecureScheme   = "wss"
	eventChanSize    = 64 // same size as the websocket outbox
	logSourceAgent   = "agent"
)

// MasterWebsocket is the type for a websocket which communicates with the master.
type MasterWebsocket = ws.WebSocket[*aproto.AgentMessage, *aproto.MasterMessage]

// Agent is the manager for all other routines in the agent. It launches the container
// manager, and all external connections, to Docker and the master. Once launched, it takes actions
// directed by the master and monitors the subroutines for failure. The agent fails and enters the
// recovery flow on the failure of any component.
type Agent struct {
	version string
	opts    options.Options
	log     *logrus.Entry
	wg      errgroupx.Group
}

// NewAgent constructs and runs a new agent according to the provided configuration.
func NewAgent(parent context.Context, version string, opts options.Options) *Agent {
	a := &Agent{
		version: version,
		opts:    opts,
		log:     logrus.WithField("component", "agent"),
		wg:      errgroupx.WithContext(parent),
	}

	a.wg.Go(func(ctx context.Context) error {
		switch err := a.run(ctx); {
		case errors.Is(err, context.Canceled):
			return nil
		case err != nil:
			return err
		default:
			return nil
		}
	})

	return a
}

// Wait for the agent to exit, returning an error indicating the reason.
func (a *Agent) Wait() error {
	return a.wg.Wait()
}

// Run sets up the agent and starts the watch loop. All configurations and system depenencies should
// be setup _before_ the watch loop is started.
//
//nolint:maintidx
func (a *Agent) run(ctx context.Context) error {
	a.log.Trace("connecting to master")
	socket, err := a.connect(ctx, false)
	if err != nil {
		return masterConnectionError{cause: fmt.Errorf("initial connection to master failed: %w", err)}
	}
	defer func() {
		a.log.Trace("cleaning up socket")
		if cErr := socket.Close(); err != nil {
			a.log.WithError(cErr).Error("failed to close master websocket")
		}
	}()

	a.log.Trace("reading master set agent options message")
	var mopts aproto.MasterSetAgentOptions
	select {
	case msg, ok := <-socket.Inbox:
		switch {
		case !ok:
			return fmt.Errorf("socket closed while reading setup messages")
		case msg.MasterSetAgentOptions == nil:
			return fmt.Errorf("master did not send setup messages")
		default:
			mopts = *msg.MasterSetAgentOptions
		}
	case <-ctx.Done():
		return fmt.Errorf("canceled while reading setup messages: %w", ctx.Err())
	}

	a.log.Trace("detecting devices")
	devices, err := detect.Detect(
		a.opts.SlotType, a.opts.AgentID, a.opts.VisibleGPUs, a.opts.ArtificialSlots,
	)
	if err != nil {
		return fmt.Errorf("failed to detect devices: %v", devices)
	}

	a.log.Tracef("setting up %s runtime", a.opts.ContainerRuntime)
	var cruntime container.ContainerRuntime
	switch a.opts.ContainerRuntime {
	case options.PodmanContainerRuntime:
		acl, sErr := podman.New(a.opts)
		if sErr != nil {
			return fmt.Errorf("failed to build podman client: %w", sErr)
		}
		defer func() {
			if cErr := acl.Close(); cErr != nil {
				a.log.WithError(cErr).Error("failed to close podman client")
			}
		}()
		cruntime = acl
	case options.ApptainerContainerRuntime:
		fallthrough
	case options.SingularityContainerRuntime:
		acl, sErr := singularity.New(a.opts)
		if sErr != nil {
			return fmt.Errorf("failed to build singularity client: %w", sErr)
		}
		defer func() {
			if cErr := acl.Close(); cErr != nil {
				a.log.WithError(cErr).Error("failed to close singularity client")
			}
		}()
		cruntime = acl
	case options.DockerContainerRuntime:
		dcl, dErr := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
		if dErr != nil {
			return fmt.Errorf("failed to build docker client: %w", dErr)
		}
		defer func() {
			a.log.Trace("cleaning up docker client")
			if cErr := dcl.Close(); cErr != nil {
				a.log.WithError(cErr).Error("failed to close docker client")
			}
		}()
		cl := docker.NewClient(dcl)
		cruntime = cl
	}

	a.log.Trace("setting up container manager")
	outbox := make(chan *aproto.MasterMessage, eventChanSize) // covers many from socket lifetimes
	manager, err := containers.New(a.opts, mopts, devices, cruntime, a.sender(outbox))
	if err != nil {
		return fmt.Errorf("error initializing container manager: %w", err)
	}
	defer func() {
		a.log.Trace("detaching container manager")
		manager.Detach()
	}()

	a.log.Trace("reattaching containers")
	reattached, err := manager.ReattachContainers(ctx, mopts.ContainersToReattach)
	if err != nil {
		return fmt.Errorf("failed to reattach containers: %w", err)
	}

	a.log.Trace("writing agent started message")
	select {
	case socket.Outbox <- &aproto.MasterMessage{AgentStarted: &aproto.AgentStarted{
		Version:              a.version,
		Devices:              devices,
		ContainersReattached: reattached,
	}}:
	case <-ctx.Done():
		return ctx.Err()
	}

	a.log.Trace("watching for ws requests and system events")
	inbox := socket.Inbox
	for {
		select {
		case msg, ok := <-inbox:
			if !ok {
				a.log.Trace("websocket inbox closed")
				inbox = nil
				continue
			}

			switch {
			case msg.StartContainer != nil:
				if err := manager.StartContainer(ctx, *msg.StartContainer); err != nil {
					a.log.WithError(err).Error("could not start container")
				}
			case msg.SignalContainer != nil:
				manager.SignalContainer(ctx, *msg.SignalContainer)
			case msg.AgentShutdown != nil:
				return errors.New(msg.AgentShutdown.ErrMsg)
			default:
				panic(fmt.Sprintf("unknown message received: %+v", msg))
			}

		case msg := <-outbox:
			select {
			case socket.Outbox <- msg:
			case <-ctx.Done():
				return nil
			}

		case <-socket.Done:
			if err := socket.Error(); err != nil {
				a.log.WithError(err).Error("socket disconnected")
			} else {
				a.log.Trace("socket disconnected")
			}

			newSocket, newMopts, err := a.reconnectFlow(ctx, manager, devices, outbox)
			if err != nil {
				return err
			}
			socket = newSocket
			inbox = socket.Inbox
			mopts = *newMopts

		case <-ctx.Done():
			a.log.Trace("context canceled")
			return nil
		}
	}
}

func (a *Agent) connect(ctx context.Context, reconnect bool) (*MasterWebsocket, error) {
	tlsConfig, err := a.tlsConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct TLS config")
	}

	masterProto := wsInsecureScheme
	if tlsConfig != nil {
		masterProto = wsSecureScheme
	}
	dialer := websocket.Dialer{
		Proxy:            websocket.DefaultDialer.Proxy,
		HandshakeTimeout: websocket.DefaultDialer.HandshakeTimeout,
		TLSClientConfig:  tlsConfig,
	}

        hostname, err := os.Hostname()

        if err != nil {
                a.log.Warnf("Unable to get hostname : %v", err)
        }

        masterAddr := fmt.Sprintf(
                "%s://%s:%d/agents?id=%s&version=%s&resource_pool=%s&reconnect=%v&hostname=%s",
                masterProto, a.opts.MasterHost, a.opts.MasterPort, a.opts.AgentID, a.version,
                a.opts.ResourcePool, reconnect, hostname,
        )
	a.log.Infof("connecting to master at: %s", masterAddr)
	conn, resp, err := dialer.DialContext(ctx, masterAddr, nil)
	if resp != nil {
		defer func() {
			if err = resp.Body.Close(); err != nil {
				a.log.WithError(err).Error("failed to close master response on connection")
			}
		}()
	}
	if err != nil {
		if resp == nil {
			return nil, errors.Wrap(err, "error dialing master")
		}

		b, rErr := io.ReadAll(resp.Body)
		if rErr == nil && strings.Contains(string(b), aproto.ErrAgentMustReconnect.Error()) {
			return nil, aproto.ErrAgentMustReconnect
		}

		return nil, errors.Wrapf(err, "error dialing master: %s", b)
	}
	return ws.Wrap[*aproto.AgentMessage, *aproto.MasterMessage](a.opts.AgentID, conn)
}

func (a *Agent) sender(out chan *aproto.MasterMessage) events.Publisher[container.Event] {
	return events.FuncPublisher[container.Event](
		func(ctx context.Context, in container.Event) error {
			var msg aproto.MasterMessage
			switch {
			case in.StateChange != nil:
				msg.ContainerStateChanged = in.StateChange
			case in.StatsRecord != nil:
				msg.ContainerStatsRecord = in.StatsRecord
			case in.Log != nil:
				msg.ContainerLog = a.enrichLog(in.Log)
			default:
				panic(fmt.Sprintf("unknown outgoing message: %+v", in))
			}

			select {
			case out <- &msg:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	)
}

func (a *Agent) enrichLog(log *aproto.ContainerLog) *aproto.ContainerLog {
	log.AgentID = &a.opts.AgentID
	if log.Source == nil {
		source := logSourceAgent
		log.Source = &source
	}
	return log
}

func (a *Agent) reconnectFlow(
	ctx context.Context,
	manager *containers.Manager,
	devices []device.Device,
	outbox chan *aproto.MasterMessage,
) (
	*MasterWebsocket,
	*aproto.MasterSetAgentOptions,
	error,
) {
	a.log.Trace("reconnecting master socket...")
	socket, err := a.reconnect(ctx)
	if err != nil {
		return nil, nil, err
	}

	a.log.Trace("reading master set agent options message after reconnect")
	var mopts *aproto.MasterSetAgentOptions
	select {
	case msg, ok := <-socket.Inbox:
		switch {
		case !ok:
			return nil, nil, fmt.Errorf("socket closed while reading setup messages")
		case msg.MasterSetAgentOptions == nil:
			return nil, nil, fmt.Errorf("master did not send setup messages")
		default:
			mopts = msg.MasterSetAgentOptions
		}
	case <-ctx.Done():
		return nil, nil, fmt.Errorf("canceled while reading setup messages: %w", ctx.Err())
	}

	a.log.Tracef("reattaching containers after reconnect: %+v", mopts.ContainersToReattach)
	reattached, err := manager.RevalidateContainers(ctx, mopts.ContainersToReattach)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to reattach containers: %w", err)
	}
	reattachedStates := make(map[cproto.ID]cproto.State)
	for _, ack := range reattached {
		reattachedStates[ack.Container.ID] = ack.Container.State
	}
	a.log.Tracef("reattached containers after reconnect: %+v", reattachedStates)

	a.log.Trace("writing agent started message")
	select {
	case socket.Outbox <- &aproto.MasterMessage{AgentStarted: &aproto.AgentStarted{
		Version:              a.version,
		Devices:              devices,
		ContainersReattached: reattached,
	}}:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}

	a.log.Trace("sending sentinel message into output stream")
	a.wg.Go(func(ctx context.Context) error {
		select {
		case outbox <- nil:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	a.log.Trace("flushing valid messages from before sentinel")
	for {
		select {
		case msg := <-outbox:
			if msg == nil {
				a.log.Trace("reconnected successfully")
				return socket, mopts, nil
			}

			if csc := msg.ContainerStateChanged; csc != nil {
				reattachState, ok := reattachedStates[msg.ContainerStateChanged.Container.ID]
				if ok && csc.Container.State.Before(reattachState) {
					a.log.Tracef(
						"dropping %s transition message for %s",
						csc.Container.ID, csc.Container.State,
					)
					continue
				}
			}

			select {
			case socket.Outbox <- msg:
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}
}

func (a *Agent) reconnect(ctx context.Context) (*MasterWebsocket, error) {
	for i := 1; ; i++ {
		switch ws, err := a.connect(ctx, true); {
		case err == nil:
			return ws, nil
		case errors.Is(err, aproto.ErrAgentMustReconnect):
			a.log.Warn("received ErrAgentMustReconnect, exiting")
			return nil, err
		case i == a.opts.AgentReconnectAttempts:
			a.log.WithError(err).Warn("exhausted reconnect attempts")
			return nil, masterConnectionError{cause: err}
		default:
			a.log.WithError(err).Error("error reconnecting to master")
		}

		select {
		case <-time.After(time.Duration(a.opts.AgentReconnectBackoff) * time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (a *Agent) tlsConfig() (*tls.Config, error) {
	if !a.opts.Security.TLS.Enabled {
		return nil, nil
	}

	var pool *x509.CertPool
	if certFile := a.opts.Security.TLS.MasterCert; certFile != "" {
		certData, err := os.ReadFile(certFile) //nolint:gosec
		if err != nil {
			msg := fmt.Sprintf("failed to read certificate file %q", certFile)
			return nil, errors.Wrapf(err, msg)
		}
		pool = x509.NewCertPool()
		if !pool.AppendCertsFromPEM(certData) {
			return nil, errors.New("certificate file contains no certificates")
		}
	}

	var certs []tls.Certificate
	switch cert, err := a.opts.Security.TLS.ReadClientCertificate(); {
	case err != nil:
		msg := fmt.Sprintf(
			"failed to read agent certificate file %q or certificate key %q",
			a.opts.Security.TLS.ClientCert, a.opts.Security.TLS.ClientKey,
		)
		return nil, errors.Wrapf(err, msg)
	case cert != nil:
		certs = append(certs, *cert)
	}

	return &tls.Config{
		InsecureSkipVerify: a.opts.Security.TLS.SkipVerify, //nolint:gosec
		MinVersion:         tls.VersionTLS12,
		RootCAs:            pool,
		ServerName:         a.opts.Security.TLS.MasterCertName,
		Certificates:       certs,
	}, nil
}

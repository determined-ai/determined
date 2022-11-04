package internal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"

	dclient "github.com/docker/docker/client"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/internal/containers"
	"github.com/determined-ai/determined/agent/internal/detect"
	"github.com/determined-ai/determined/agent/internal/fluent"
	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/groupx/errgroupx"
	"github.com/determined-ai/determined/master/pkg/ws"
)

const (
	wsInsecureScheme = "ws"
	wsSecureScheme   = "wss"
	eventChanSize    = 64 // same size as the websocket outbox
)

// MasterWebsocket is the type for a websocket which communicates with the master.
type MasterWebsocket = ws.Websocket[*aproto.AgentMessage, *aproto.MasterMessage]

// Agent is the manager for all other processes in the agent. It launches Fluent Bit, the container
// manager, and all external connections, to Docker and the master. Once launched, it takes actions
// directed by the master and monitors the subprocesses for failure. The agent fails and enters the
// recovery flow on the failure of any component.
type Agent struct {
	// Configuration details
	version string
	opts    options.AgentOptions
	mopts   aproto.MasterSetAgentOptions
	devices []device.Device

	// System dependencies
	log     *logrus.Entry
	manager *containers.Manager
	socket  *MasterWebsocket
	docker  *docker.Client
	fluent  *fluent.Fluent

	// Internal state
	outbox chan container.Event

	wg errgroupx.Group
}

// New constructs and runs a new agent according to the provided configuration.
func New(parent context.Context, version string, options options.AgentOptions) *Agent {
	a := &Agent{
		version: version,
		opts:    options,

		log: logrus.WithField("component", "agent"),

		wg: errgroupx.WithContext(parent),
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
func (a *Agent) run(ctx context.Context) (err error) {
	a.log.Trace("detecting devices")
	a.devices, err = detect.Detect(
		a.opts.SlotType, a.opts.AgentID, a.opts.VisibleGPUs, a.opts.ArtificialSlots,
	)
	if err != nil {
		return fmt.Errorf("failed to detect devices: %v", a.devices)
	}

	a.log.Trace("connecting to master")
	a.socket, err = a.connect(ctx)
	if err != nil {
		return fmt.Errorf("crashing due to websocket connection failure: %w", err)
	}
	defer func() {
		a.log.Trace("cleaning up socket")
		if cErr := a.socket.Close(); err != nil {
			a.log.WithError(cErr).Error("closing master websocket")
		}
	}()

	a.log.Trace("reading master set agent options message")
	select {
	case msg, ok := <-a.socket.Inbox:
		switch {
		case !ok:
			return fmt.Errorf("socket closed while reading setup messages")
		case msg.MasterSetAgentOptions == nil:
			return fmt.Errorf("master did not send setup messages")
		default:
			a.mopts = *msg.MasterSetAgentOptions
		}
	case <-ctx.Done():
		return fmt.Errorf("canceled while reading setup messages: %w", ctx.Err())
	}

	a.log.Trace("setting up docker client")
	cl, err := dclient.NewClientWithOpts(dclient.WithAPIVersionNegotiation(), dclient.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to build docker client: %w", err)
	}
	a.docker = docker.NewClient(cl)
	defer func() {
		a.log.Trace("cleaning up docker client")
		if cErr := a.docker.Close(); err != nil {
			a.log.WithError(cErr).Error("closing docker client")
		}
	}()

	a.log.Trace("setting up fluentbit daemon")
	a.fluent, err = fluent.Start(ctx, a.opts, a.mopts, a.docker)
	if err != nil {
		return fmt.Errorf("setting up fluentbit failed: %w", err)
	}
	defer func() {
		a.log.Trace("cleaning up fluent client")
		if cErr := a.fluent.Close(); err != nil {
			a.log.WithError(cErr).Error("closing fluentbit")
		}
	}()

	a.log.Trace("setting up container manager")
	a.manager, err = containers.New(a.opts, a.mopts, a.devices, a.docker, a.sender())
	if err != nil {
		return fmt.Errorf("error initializing container manager: %w", err)
	}
	defer func() {
		a.log.Trace("detaching container manager")
		a.manager.Detach()
	}()

	a.log.Trace("reattaching containers")
	reattached, err := a.manager.ReattachContainers(ctx, a.mopts.ContainersToReattach)
	if err != nil {
		return fmt.Errorf("failed to reattach containers: %w", err)
	}

	a.log.Trace("writing agent started message")
	select {
	case a.socket.Outbox <- &aproto.MasterMessage{AgentStarted: &aproto.AgentStarted{
		Version:              a.version,
		Devices:              a.devices,
		Label:                a.opts.Label,
		ContainersReattached: reattached,
	}}:
	case <-ctx.Done():
		return ctx.Err()
	}

	return a.watch(ctx)
}

func (a *Agent) watch(ctx context.Context) error {
	a.log.Trace("watching for ws requests and system events")
	inbox := a.socket.Inbox
	outbox := make(chan *aproto.MasterMessage, eventChanSize)
	for {
		select {
		case msg, ok := <-inbox:
			a.log.Tracef("received message: %v", msg)
			if !ok {
				return errors.New("crashing due to websocket inbox closure")
			}
			if err := a.receive(ctx, msg); err != nil {
				return err
			}

		case msg := <-outbox:
			a.log.Tracef("sent message: %v", msg)
			select {
			case a.socket.Outbox <- msg:
			case <-ctx.Done():
				return nil
			}

		case <-a.socket.Done:
			a.log.Trace("socket exited")
			if err := a.socket.Error(); err != nil {
				return fmt.Errorf("crashing due to websocket connection failure: %w", err)
			}
			return nil

		case <-a.fluent.Done:
			a.log.Trace("fluent exited")
			if err := a.fluent.Error(); err != nil {
				return fmt.Errorf("restarting due fluent failure: %w", err)
			}
			return nil

		case <-ctx.Done():
			a.log.Trace("context canceled")
			return nil
		}
	}
}

func (a *Agent) send(ctx context.Context, msg *aproto.MasterMessage) error {
	select {
	case a.socket.Outbox <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *Agent) receive(ctx context.Context, msg *aproto.AgentMessage) error {
	switch {
	case msg.StartContainer != nil:
		if err := a.manager.StartContainer(ctx, *msg.StartContainer); err != nil {
			a.log.WithError(err).Error("could not starting container")
		}
	case msg.SignalContainer != nil:
		a.manager.SignalContainer(ctx, *msg.SignalContainer)
	default:
		panic(fmt.Sprintf("unknown message received: %+v", msg))
	}
	return nil
}

func (a *Agent) sender() events.Publisher[container.Event] {
	return events.FuncPublisher[container.Event](
		func(ctx context.Context, in container.Event) error {
			var out aproto.MasterMessage
			switch {
			case in.StateChange != nil:
				out.ContainerStateChanged = in.StateChange
			case in.StatsRecord != nil:
				out.ContainerStatsRecord = in.StatsRecord
			case in.Log != nil:
				out.ContainerLog = in.Log
			default:
				panic(fmt.Sprintf("unknown outgoing message: %+v", in))
			}
			return a.send(ctx, &out)
		},
	)
}

func (a *Agent) connect(ctx context.Context) (*MasterWebsocket, error) {
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

	masterAddr := fmt.Sprintf(
		"%s://%s:%d/agents?id=%s&version=%s&resource_pool=%s&reconnect=%v",
		masterProto, a.opts.MasterHost, a.opts.MasterPort, a.opts.AgentID, a.version,
		a.opts.ResourcePool, false,
	)
	a.log.Infof("connecting to master at: %s", masterAddr)
	conn, resp, err := dialer.DialContext(ctx, masterAddr, nil)
	if resp != nil {
		defer func() {
			if err = resp.Body.Close(); err != nil {
				a.log.WithError(err).Error("failed to read master response on connection")
			}
		}()
	}
	if err != nil {
		if resp == nil {
			return nil, errors.Wrap(err, "error dialing master")
		}

		b, rErr := ioutil.ReadAll(resp.Body)
		if rErr == nil && strings.Contains(string(b), aproto.ErrAgentMustReconnect.Error()) {
			return nil, aproto.ErrAgentMustReconnect
		}

		return nil, errors.Wrapf(err, "error dialing master: %s", b)
	}
	return ws.Wrap[*aproto.AgentMessage, *aproto.MasterMessage](a.opts.AgentID, conn), nil
}

func (a *Agent) tlsConfig() (*tls.Config, error) {
	if !a.opts.Security.TLS.Enabled {
		return nil, nil
	}

	var pool *x509.CertPool
	if certFile := a.opts.Security.TLS.MasterCert; certFile != "" {
		certData, err := ioutil.ReadFile(certFile) //nolint:gosec
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

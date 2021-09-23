package internal

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	proto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	httpInsecureScheme = "http"
	httpSecureScheme   = "https"
	wsInsecureScheme   = "ws"
	wsSecureScheme     = "wss"
)

type agent struct {
	Version               string
	Options               `json:"options"`
	MasterSetAgentOptions *proto.MasterSetAgentOptions
	Devices               []device.Device `json:"devices"`

	socket *actor.Ref
	cm     *actor.Ref
	fluent *actor.Ref

	masterProto  string
	masterClient *http.Client

	reconnecting bool
}

func newAgent(version string, options Options) *agent {
	return &agent{Version: version, Options: options}
}

func (a *agent) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Log().Infof("Determined agent %s (built with %s)", a.Version, runtime.Version())
		actors.NotifyOnSignal(ctx, syscall.SIGINT, syscall.SIGTERM)
		return a.connect(ctx)

	case proto.AgentMessage:
		switch {
		case msg.MasterSetAgentOptions != nil:
			if a.MasterSetAgentOptions != nil {
				ctx.Log().Debugf("received MasterStepAgentOptions more than once: %v",
					*msg.MasterSetAgentOptions)
				return nil
			}
			a.MasterSetAgentOptions = msg.MasterSetAgentOptions
			return a.setup(ctx)
		case msg.StartContainer != nil:
			a.addProxy(&msg.StartContainer.Spec.RunSpec.ContainerConfig)
			if !a.validateDevices(msg.StartContainer.Container.Devices) {
				return errors.New("could not start container; devices specified in spec not found on agent")
			}
			ctx.Tell(a.cm, *msg.StartContainer)
		case msg.SignalContainer != nil:
			ctx.Tell(a.cm, *msg.SignalContainer)
		default:
			panic(fmt.Sprintf("unknown message received: %+v", msg))
		}

	case proto.ContainerStateChanged:
		if a.socket != nil {
			ctx.Ask(a.socket, api.WriteMessage{Message: proto.MasterMessage{ContainerStateChanged: &msg}})
		} else {
			ctx.Log().Warnf("Not sending container state change to the master: %+v", msg)
		}
	case proto.ContainerLog:
		if a.socket != nil {
			ctx.Ask(a.socket, api.WriteMessage{Message: proto.MasterMessage{ContainerLog: &msg}})
		}

	case model.TrialLog:
		return a.postTrialLog(msg)

	case actor.ChildFailed:
		switch msg.Child {
		case a.socket:
			if a.attemptReconnect(ctx) {
				return nil
			}
			ctx.Log().Warn("master socket disconnected, shutting down agent...")
		case a.cm:
			ctx.Log().Warn("container manager failed, shutting down agent...")
		case a.fluent:
			ctx.Log().Warn("fluent bit failed, restarting it...")
			go func() {
				// Do this in a goroutine so we don't block the agent actor while retrying.
				if err := a.restartFluent(ctx); err != nil {
					logrus.WithError(err).Error("failed to restart fluent with retries")
					ctx.Self().Stop()
				}
			}()
			return nil
		}
		return errors.Wrapf(msg.Error, "unexpected child failure: %s", msg.Child.Address())

	case actor.ChildStopped:
		return errors.Errorf("unexpected child stopped: %s", msg.Child.Address())

	case os.Signal:
		switch msg {
		case syscall.SIGINT, syscall.SIGTERM:
			ctx.Log().Info("shutting down agent...")
			ctx.Self().Stop()
		default:
			ctx.Log().Infof("unexpected signal received: %s", msg)
		}

	case echo.Context:
		a.handleAPIRequest(ctx, msg)

	case actor.PostStop:
		if a.fluent != nil {
			if err := a.fluent.StopAndAwaitTermination(); err != nil {
				ctx.Log().Errorf("error killing logging container %v", err)
			}
		}

		ctx.Log().Info("agent shut down")

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (a *agent) restartFluent(ctx *actor.Context) error {
	i := 0
	for {
		fluentActor, err := newFluentActor(a.Options, *a.MasterSetAgentOptions)
		switch {
		case err == nil:
			a.fluent, _ = ctx.ActorOf("fluent", fluentActor)
			return nil
		case err != nil && i >= 5:
			return errors.Wrap(err, "failed to restart Fluent daemon")
		default:
			ctx.Log().Warnf("failed to restart Fluent daemon: %s", err)
			// Just use exponential backoff.
			t := time.Duration(math.Pow(2, float64(i))) * time.Second
			i++
			ctx.Log().Infof("trying to restart Fluent daemon in %s", t)
			time.Sleep(t)
		}
	}
}

func (a *agent) addProxy(config *container.Config) {
	addVars := map[string]string{
		"HTTP_PROXY":  a.Options.HTTPProxy,
		"HTTPS_PROXY": a.Options.HTTPSProxy,
		"FTP_PROXY":   a.Options.FTPProxy,
		"NO_PROXY":    a.Options.NoProxy,
	}

	for _, v := range config.Env {
		key := strings.SplitN(v, "=", 2)[0]
		key = strings.ToUpper(key)
		_, ok := addVars[key]
		if ok {
			delete(addVars, key)
		}
	}

	for k, v := range addVars {
		if v != "" {
			config.Env = append(config.Env, k+"="+v)
		}
	}
}

// validateDevices checks the devices requested in container.Spec are a subset of agent devices.
func (a *agent) validateDevices(devices []device.Device) bool {
	for _, d := range devices {
		if !a.containsDevice(d) {
			return false
		}
	}
	return true
}

func (a *agent) containsDevice(d device.Device) bool {
	for _, dev := range a.Devices {
		if d.ID == dev.ID {
			return true
		}
	}
	return false
}

func (a *agent) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		ctx.Respond(apiCtx.JSON(http.StatusOK, a))

	case echo.POST:
		body, err := ioutil.ReadAll(apiCtx.Request().Body)
		if err != nil {
			ctx.Respond(err)
			return
		}

		var msg proto.AgentMessage
		if err = json.Unmarshal(body, &msg); err != nil {
			ctx.Respond(err)
			return
		}

		switch {
		case msg.StartContainer != nil:
			switch result := ctx.Ask(a.cm, *msg.StartContainer).Get().(type) {
			case error:
				ctx.Respond(err)
			default:
				ctx.Respond(apiCtx.JSON(http.StatusOK, result))
			}
		case msg.SignalContainer != nil:
			ctx.Tell(a.cm, *msg.SignalContainer)
		default:
			ctx.Respond(errors.Errorf("unknown message received"))
		}

	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (a *agent) tlsConfig() (*tls.Config, error) {
	if !a.Options.Security.TLS.Enabled {
		return nil, nil
	}

	var pool *x509.CertPool
	if certFile := a.Options.Security.TLS.MasterCert; certFile != "" {
		certData, err := ioutil.ReadFile(certFile) //nolint:gosec
		if err != nil {
			return nil, errors.Wrap(err, "failed to read certificate file")
		}
		pool = x509.NewCertPool()
		if !pool.AppendCertsFromPEM(certData) {
			return nil, errors.New("certificate file contains no certificates")
		}
	}

	return &tls.Config{
		InsecureSkipVerify: a.Options.Security.TLS.SkipVerify, //nolint:gosec
		MinVersion:         tls.VersionTLS12,
		RootCAs:            pool,
		ServerName:         a.Options.Security.TLS.MasterCertName,
	}, nil
}

func (a *agent) makeMasterClient() error {
	tlsConfig, err := a.tlsConfig()
	if err != nil {
		return errors.Wrap(err, "failed to construct TLS config")
	}

	a.masterProto = httpInsecureScheme
	if tlsConfig != nil {
		a.masterProto = httpSecureScheme
	}
	a.masterClient = &http.Client{
		Transport: &http.Transport{
			Proxy:           http.DefaultTransport.(*http.Transport).Proxy,
			TLSClientConfig: tlsConfig,
		},
	}
	return nil
}

func (a *agent) makeMasterWebsocket(ctx *actor.Context) error {
	tlsConfig, err := a.tlsConfig()
	if err != nil {
		return errors.Wrap(err, "failed to construct TLS config")
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

	masterAddr := fmt.Sprintf("%s://%s:%d/agents?id=%s&resource_pool=%s&reconnect=%t",
		masterProto, a.MasterHost, a.MasterPort, a.AgentID, a.ResourcePool, a.reconnecting)
	ctx.Log().Infof("connecting to master at: %s", masterAddr)
	conn, resp, err := dialer.Dial(masterAddr, nil)
	if resp != nil {
		defer func() {
			if err = resp.Body.Close(); err != nil {
				ctx.Log().WithError(err).Error("failed to read master response on connection")
			}
		}()
	}
	if err != nil {
		if resp == nil {
			return errors.Wrap(err, "error dialing master")
		}

		b, rErr := ioutil.ReadAll(resp.Body)
		if rErr == nil && strings.Contains(string(b), proto.ErrAgentMustReconnect.Error()) {
			return proto.ErrAgentMustReconnect
		}

		return errors.Wrapf(err, "error dialing master: %s", b)
	}

	a.socket, _ = ctx.ActorOf("websocket", api.WrapSocket(conn, proto.AgentMessage{}, true))
	return nil
}

func (a *agent) attemptReconnect(ctx *actor.Context) bool {
	a.reconnecting = true
	defer func() {
		a.reconnecting = false
	}()
	for i := 0; i < proto.AgentReconnectAttempts; i++ {
		switch err := a.connect(ctx); {
		case err == nil:
			return true
		case errors.Is(err, proto.ErrAgentMustReconnect):
			return false
		default:
			ctx.Log().WithError(err).Error("error to reconnecting to master")
		}
		time.Sleep(proto.AgentReconnectBackoff)
	}
	return false
}

func (a *agent) connect(ctx *actor.Context) error {
	if a.MasterPort == 0 {
		if a.Options.Security.TLS.Enabled {
			a.MasterPort = 443
		} else {
			a.MasterPort = 80
		}
	}

	if a.MasterHost != "" {
		if err := a.connectToMaster(ctx); err != nil {
			return err
		}
		ctx.Log().Infof("successfully connected to master")
	} else {
		ctx.Log().Warn("no master address specified; running in standalone mode")
	}
	return nil
}

func (a *agent) setup(ctx *actor.Context) error {
	fluentActor, err := newFluentActor(a.Options, *a.MasterSetAgentOptions)
	if err != nil {
		return errors.Wrap(err, "failed to start Fluent daemon")
	}
	a.fluent, _ = ctx.ActorOf("fluent", fluentActor)

	if err = a.detect(); err != nil {
		return err
	}
	ctx.Log().Info("detected compute devices:")
	for _, d := range a.Devices {
		ctx.Log().Infof("\t%s", d.String())
	}

	v, err := getNvidiaVersion()
	if err != nil {
		return err
	} else if v != "" {
		ctx.Log().Infof("Nvidia driver version: %s", v)
	}

	if a.MasterPort == 0 {
		if a.Options.Security.TLS.Enabled {
			a.MasterPort = 443
		} else {
			a.MasterPort = 80
		}
	}

	cm, err := newContainerManager(a, fluentActor.port)
	if err != nil {
		return errors.Wrap(err, "error initializing container manager")
	}
	a.cm, _ = ctx.ActorOf("containers", cm)

	ctx.Ask(a.socket, api.WriteMessage{Message: proto.MasterMessage{AgentStarted: &proto.AgentStarted{
		Version: a.Version,
		Devices: a.Devices,
		Label:   a.Label,
	}}})
	return nil
}

func (a *agent) connectToMaster(ctx *actor.Context) error {
	if err := a.makeMasterClient(); err != nil {
		return errors.Wrap(err, "error creating master client")
	}
	if err := a.makeMasterWebsocket(ctx); err != nil {
		return errors.Wrap(err, "error connecting to master")
	}
	return nil
}

func (a *agent) postTrialLog(log model.TrialLog) error {
	j, err := json.Marshal([]model.TrialLog{log})
	if err != nil {
		return err
	}

	resp, err := a.masterClient.Post(
		fmt.Sprintf("%s://%s:%d/trial_logs", a.masterProto, a.MasterHost, a.MasterPort),
		"application/json",
		bytes.NewReader(j),
	)
	if err != nil {
		return errors.Wrap(err, "failed to post trial log")
	}
	if err := resp.Body.Close(); err != nil {
		return errors.Wrap(err, "failed to read master response for trial log")
	}
	return nil
}

func runAPIServer(options Options, system *actor.System) error {
	server := echo.New()
	server.Logger = logger.New()
	server.HidePort = true
	server.HideBanner = true
	server.Use(middleware.Recover())
	server.Pre(middleware.RemoveTrailingSlash())

	server.Any("/*", api.Route(system, nil))
	server.Any("/debug/pprof/*", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	server.Any("/debug/pprof/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	server.Any("/debug/pprof/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	server.Any("/debug/pprof/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	server.Any("/debug/pprof/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))

	bindAddr := fmt.Sprintf("%s:%d", options.BindIP, options.BindPort)
	logrus.Infof("starting agent server on [%s]", bindAddr)
	if options.TLS {
		return server.StartTLS(bindAddr, options.CertFile, options.KeyFile)
	}
	return server.Start(bindAddr)
}

// Run runs a new agent system and actor with the provided options.
func Run(version string, options Options) error {
	printableConfig, err := options.Printable()
	if err != nil {
		return err
	}
	logrus.Infof("agent configuration: %s", printableConfig)

	system := actor.NewSystem(options.AgentID)
	ref, _ := system.ActorOf(actor.Addr("agent"), newAgent(version, options))

	errs := make(chan error)
	if options.APIEnabled {
		go func() {
			errs <- runAPIServer(options, system)
		}()
	}
	go func() {
		errs <- ref.AwaitTermination()
	}()
	return <-errs
}

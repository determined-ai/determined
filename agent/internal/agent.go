package internal

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"syscall"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/agent/version"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	proto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
)

type agent struct {
	Options    `json:"options"`
	Devices    []device.Device  `json:"devices"`
	MasterInfo proto.MasterInfo `json:"master"`

	socket *actor.Ref
	cm     *actor.Ref
}

func (a *agent) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		return a.setup(ctx)

	case proto.AgentMessage:
		switch {
		case msg.StartContainer != nil:
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

	case actor.ChildFailed:
		switch msg.Child {
		case a.socket:
			ctx.Log().Warn("master socket disconnected, shutting down agent...")
		case a.cm:
			ctx.Log().Warn("container manager failed, shutting down agent...")
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
		ctx.Log().Info("agent shut down")

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
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

func (a *agent) getMasterInfo() error {
	masterProto := "http"
	if a.TLS {
		masterProto = "https"
	}
	resp, err := http.Get(fmt.Sprintf("%s://%s:%d/info", masterProto, a.MasterHost, a.MasterPort))
	if err != nil {
		return errors.Wrap(err, "failed to get master info")
	}
	if err = json.NewDecoder(resp.Body).Decode(&a.MasterInfo); err != nil {
		return errors.Wrap(err, "failed to read master info")
	}
	if err := resp.Body.Close(); err != nil {
		return errors.Wrap(err, "failed to read master response on connection")
	}
	return nil
}

func (a *agent) setup(ctx *actor.Context) error {
	ctx.Log().Infof("Determined agent %s (built with %s)", version.Version, runtime.Version())
	actors.NotifyOnSignal(ctx, syscall.SIGINT, syscall.SIGTERM)

	if a.ArtificialSlots > 0 {
		for i := 0; i < a.ArtificialSlots; i++ {
			id := uuid.New().String()
			a.Devices = append(a.Devices, device.Device{
				ID: i, Brand: "Artificial", UUID: id, Type: device.CPU})
		}
	} else {
		d, err := detectDevices(a.Options.VisibleGPUs)
		if err != nil {
			return err
		}
		a.Devices = d
	}
	ctx.Log().Info("detected compute devices:")
	for _, d := range a.Devices {
		ctx.Log().Infof("\t%s", d.String())
	}

	if v, err := getNvidiaVersion(); err != nil {
		return err
	} else if v != "" {
		ctx.Log().Infof("Nvidia driver version: %s", v)
	}

	if a.MasterPort == 0 {
		if a.TLS {
			a.MasterPort = 443
		} else {
			a.MasterPort = 80
		}
	}

	if err := a.getMasterInfo(); err != nil {
		return errors.Wrap(err, "error fetching master info")
	}

	a.cm, _ = ctx.ActorOf("containers",
		&containerManager{MasterInfo: a.MasterInfo, Options: a.Options, Devices: a.Devices})

	if a.MasterHost != "" {
		if err := a.connectToMaster(ctx); err != nil {
			return err
		}
	} else {
		ctx.Log().Warn("no master address specified; running in standalone mode")
	}

	return nil
}

func (a *agent) connectToMaster(ctx *actor.Context) error {
	dialer := websocket.Dialer{
		Proxy:            websocket.DefaultDialer.Proxy,
		HandshakeTimeout: websocket.DefaultDialer.HandshakeTimeout,
	}
	masterProto := "ws"
	if a.TLS {
		masterProto = "wss"
		dialer.TLSClientConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	masterAddr := fmt.Sprintf("%s://%s:%d/agents?id=%s",
		masterProto, a.MasterHost, a.MasterPort, a.AgentID)
	ctx.Log().Infof("connecting to master at: %s", masterAddr)
	conn, resp, err := dialer.Dial(masterAddr, nil)
	if err != nil {
		return errors.Wrap(err, "error connecting to master")
	} else if err = resp.Body.Close(); err != nil {
		return errors.Wrap(err, "failed to read master response on connection")
	}
	ctx.Log().Infof("successfully connected to master")

	a.socket, _ = ctx.ActorOf("socket", api.WrapSocket(conn, proto.AgentMessage{}, true))

	containers := ctx.Ask(a.cm, recoverContainers{}).Get().([]proto.ContainerRecovered)

	started := proto.MasterMessage{AgentStarted: &proto.AgentStarted{
		Version: version.Version, Devices: a.Devices, Label: a.Label, RecoveredContainers: containers}}
	ctx.Ask(a.socket, api.WriteMessage{Message: started})
	return nil
}

// Run runs a new agent system and actor with the provided options.
func Run(options Options) error {
	printableConfig, err := options.Printable()
	if err != nil {
		return err
	}
	logrus.Infof("agent configuration: %s", printableConfig)

	system := actor.NewSystem(options.AgentID)
	ref, _ := system.ActorOf(actor.Addr("agent"), &agent{Options: options})

	server := echo.New()
	server.Logger = logger.New()
	server.HidePort = true
	server.HideBanner = true
	server.Use(middleware.Recover())
	server.Pre(middleware.RemoveTrailingSlash())

	server.Any("/*", api.Route(system))
	server.Any("/debug/pprof/*", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	server.Any("/debug/pprof/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	server.Any("/debug/pprof/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	server.Any("/debug/pprof/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	server.Any("/debug/pprof/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))

	errs := make(chan error)
	if options.APIEnabled {
		go func() {
			bindAddr := fmt.Sprintf("%s:%d", options.BindIP, options.BindPort)
			logrus.Infof("starting agent server on [%s]", bindAddr)
			if options.TLS {
				errs <- server.StartTLS(bindAddr, options.CertFile, options.KeyFile)
			} else {
				errs <- server.Start(bindAddr)
			}
		}()
	}
	go func() {
		errs <- ref.AwaitTermination()
	}()
	return <-errs
}

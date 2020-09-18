package internal

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/pprof"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/internal/template"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

const defaultAskTimeout = 2 * time.Second

// Master manages the Determined master state.
type Master struct {
	ClusterID string
	MasterID  string
	Version   string

	config          *Config
	defaultTaskSpec *tasks.TaskSpec

	logs          *logger.LogBuffer
	system        *actor.System
	echo          *echo.Echo
	rp            *actor.Ref
	rwCoordinator *actor.Ref
	provisioner   *actor.Ref
	db            *db.PgDB
	proxy         *actor.Ref
	trialLogger   *actor.Ref
}

// New creates an instance of the Determined master.
func New(version string, logStore *logger.LogBuffer, config *Config) *Master {
	logger.SetLogrus(config.Log)
	return &Master{
		MasterID: uuid.New().String(),
		Version:  version,
		logs:     logStore,
		config:   config,
	}
}

func (m *Master) getConfig(c echo.Context) (interface{}, error) {
	return m.config.Printable()
}

func (m *Master) getInfo(c echo.Context) (interface{}, error) {
	telemetryInfo := aproto.TelemetryInfo{}

	if m.config.Telemetry.Enabled && m.config.Telemetry.SegmentWebUIKey != "" {
		// Only advertise a Segment WebUI key if a key has been configured and
		// telemetry is enabled.
		telemetryInfo.Enabled = true
		telemetryInfo.SegmentKey = m.config.Telemetry.SegmentWebUIKey
	}

	return &aproto.MasterInfo{
		ClusterID: m.ClusterID,
		MasterID:  m.MasterID,
		Version:   m.Version,
		Telemetry: telemetryInfo,
	}, nil
}

func (m *Master) getMasterLogs(c echo.Context) (interface{}, error) {
	args := struct {
		LessThanID    *int `query:"less_than_id"`
		GreaterThanID *int `query:"greater_than_id"`
		Limit         *int `query:"tail"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}

	limit := -1
	if args.Limit != nil {
		limit = *args.Limit
	}

	startID := -1
	if args.GreaterThanID != nil {
		startID = *args.GreaterThanID + 1
	}

	endID := -1
	if args.LessThanID != nil {
		endID = *args.LessThanID
	}

	entries := m.logs.Entries(startID, endID, limit)
	if len(entries) == 0 {
		// Return a zero-length array here so the JSON encoding is `[]` rather than `null`.
		entries = make([]*logger.Entry, 0)
	}
	return entries, nil
}

func (m *Master) readTLSCertificate() (*tls.Certificate, error) {
	certFile := m.config.Security.TLS.Cert
	keyFile := m.config.Security.TLS.Key
	switch {
	case certFile == "" && keyFile != "":
		return nil, errors.New("TLS key was provided without a cert")
	case certFile != "" && keyFile == "":
		return nil, errors.New("TLS cert was provided without a key")
	case certFile != "" && keyFile != "":
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load TLS files")
		}
		return &cert, nil
	}
	return nil, nil
}

func (m *Master) startServers(cert *tls.Certificate) error {
	// Create the desired server configurations. The values of the servers map are descriptions to be
	// used in making error messages more informative.
	servers := make(map[*http.Server]string)

	if m.config.Security.HTTP {
		servers[&http.Server{
			Addr: fmt.Sprintf(":%d", m.config.HTTPPort),
		}] = "http server"
	}

	if cert != nil {
		servers[&http.Server{
			Addr: fmt.Sprintf(":%d", m.config.HTTPSPort),
			TLSConfig: &tls.Config{
				Certificates:             []tls.Certificate{*cert},
				MinVersion:               tls.VersionTLS12,
				PreferServerCipherSuites: true,
			},
		}] = "https server"
	}

	if len(servers) == 0 {
		return errors.New("master was not configured to listen on any port")
	}

	if err := grpc.RegisterHTTPProxy(m.echo, m.config.GRPCPort, m.config.EnableCors); err != nil {
		return errors.Wrap(err, "failed to register gRPC proxy")
	}

	// Start all servers.
	errs := make(chan error)
	defer close(errs)

	go func() {
		errs <- grpc.StartGRPCServer(m.db, &apiServer{m: m}, m.config.GRPCPort)
	}()

	for server := range servers {
		go func(server *http.Server) {
			errs <- errors.Wrap(m.echo.StartServer(server), servers[server]+" failed")
		}(server)
	}

	// Wait for all servers to terminate; return only the first error received, if any (since we close
	// all servers on error, other servers are likely to return unhelpful "server closed" errors).
	var firstErr error
	for range servers {
		if err := <-errs; err != nil && firstErr == nil {
			firstErr = err
			for server, desc := range servers {
				if cErr := server.Close(); cErr != nil {
					log.Errorf("failed to close %s: %s", desc, cErr)
				}
			}
		}
	}
	return firstErr
}

func (m *Master) restoreExperiment(e *model.Experiment) {
	// Check if the returned config is the zero value, i.e. the config could not be parsed
	// correctly. If the config could not be parsed, mark the experiment as errored.
	if !reflect.DeepEqual(e.Config, model.ExperimentConfig{}) {
		err := restoreExperiment(m, e)
		if err == nil {
			return
		}
		log.WithError(err).Errorf("failed to restore experiment: %d", e.ID)
	} else {
		log.Errorf("failed to parse experiment config: %d", e.ID)
	}
	e.State = model.ErrorState
	if err := m.db.TerminateExperimentInRestart(e.ID, e.State); err != nil {
		log.WithError(err).Error("failed to mark experiment as errored")
	}
	telemetry.ReportExperimentStateChanged(m.system, m.db, *e)
}

// convertDBErrorsToNotFound helps reduce boilerplate in our handlers, by
// classifying database "not found" errors as HTTP "not found" errors.
func convertDBErrorsToNotFound(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if errors.Cause(err) == db.ErrNotFound {
			return echo.ErrNotFound
		}
		return err
	}
}

func (m *Master) rwCoordinatorWebSocket(socket *websocket.Conn, c echo.Context) error {
	c.Logger().Infof(
		"New connection for RW Coordinator from: %v, %s",
		socket.RemoteAddr(),
		c.Request().URL,
	)

	resourceName := c.Request().URL.Path
	query := c.Request().URL.Query()

	readLockString, ok := query["read_lock"]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Sprintf("Received request without specifying read_lock: %v", c.Request().URL))
	}

	var readLock bool
	if strings.EqualFold(readLockString[0], "True") {
		readLock = true
	} else {
		if !strings.EqualFold(readLockString[0], "false") {
			return echo.NewHTTPError(http.StatusBadRequest,
				fmt.Sprintf("Received request with invalid read_lock: %v", c.Request().URL))
		}
		readLock = false
	}

	socketActor := m.system.AskAt(actor.Addr("rwCoordinator"),
		resourceRequest{resourceName, readLock, socket})
	actorRef, ok := socketActor.Get().(*actor.Ref)
	if !ok {
		c.Logger().Errorf("Failed to get websocket actor")
		return nil
	}

	// Wait for the websocket actor to terminate.
	return actorRef.AwaitTermination()
}

func (m *Master) initializeResourceProviders(provisionerSlotsPerInstance int) {
	var resourceProvider *actor.Ref
	switch {
	case m.config.Scheduler.ResourceProvider.DefaultRPConfig != nil:
		resourceProvider, _ = m.system.ActorOf(actor.Addr("defaultRP"), scheduler.NewDefaultRP(
			m.config.Scheduler.MakeScheduler(),
			m.config.Scheduler.FitFunction(),
			m.provisioner,
			provisionerSlotsPerInstance,
		))

	case m.config.Scheduler.ResourceProvider.KubernetesRPConfig != nil:
		resourceProvider, _ = m.system.ActorOf(
			actor.Addr("kubernetesRP"),
			scheduler.NewKubernetesResourceProvider(
				m.config.Scheduler.ResourceProvider.KubernetesRPConfig,
			),
		)

	default:
		panic("no expected resource provider config is defined")
	}

	m.rp, _ = m.system.ActorOf(
		actor.Addr("resourceProviders"),
		scheduler.NewResourceProviders(resourceProvider))
}

// Run causes the Determined master to connect the database and begin listening for HTTP requests.
func (m *Master) Run() error {
	log.Infof("Determined master %s (built with %s)", m.Version, runtime.Version())

	var err error

	if err = etc.SetRootPath(filepath.Join(m.config.Root, "static/srv")); err != nil {
		return errors.Wrap(err, "could not set static root")
	}

	m.db, err = db.Setup(&m.config.DB)
	if err != nil {
		return err
	}

	m.ClusterID, err = m.db.GetClusterID()
	if err != nil {
		return errors.Wrap(err, "could not fetch cluster id from database")
	}
	cert, err := m.readTLSCertificate()
	if err != nil {
		return errors.Wrap(err, "failed to read TLS certificate")
	}
	m.defaultTaskSpec = &tasks.TaskSpec{
		ClusterID:             m.ClusterID,
		HarnessPath:           filepath.Join(m.config.Root, "wheels"),
		TaskContainerDefaults: m.config.TaskContainerDefaults,
		MasterCert:            cert,
	}

	go m.cleanUpSearcherEvents()

	// Actor structure:
	// master system
	// +- Provisioner (provisioner.Provisioner: provisioner)
	// +- ResourceProviders (scheduler.ResourceProviders: resourceProviders)
	// Exactly one of the resource providers is enabled at a time.
	// +- DefaultResourceProvider (scheduler.DefaultResourceProvider: defaultRP)
	// +- KubernetesResourceProvider (scheduler.KubernetesResourceProvider: kubernetesRP)
	// +- Service Proxy (proxy.Proxy: proxy)
	// +- RWCoordinator (internal.rw_coordinator: rwCoordinator)
	// +- Telemetry (telemetry.telemetryActor: telemetry)
	// +- TrialLogger (internal.trialLogger: trialLogger)
	// +- Experiments (actors.Group: experiments)
	//     +- Experiment (internal.experiment: <experiment-id>)
	//         +- Trial (internal.trial: <trial-request-id>)
	//             +- Websocket (actors.WebSocket: <remote-address>)
	// +- Agent Group (actors.Group: agents)
	//     +- Agent (internal.agent: <agent-id>)
	//         +- Websocket (actors.WebSocket: <remote-address>)
	m.system = actor.NewSystem("master")

	m.trialLogger, _ = m.system.ActorOf(actor.Addr("trialLogger"), newTrialLogger(m.db))

	userService, err := user.New(m.db, m.system)
	if err != nil {
		return errors.Wrap(err, "cannot initialize user manager")
	}
	authFuncs := []echo.MiddlewareFunc{userService.ProcessAuthentication}

	var p *provisioner.Provisioner
	p, m.provisioner, err = provisioner.Setup(m.system, m.config.Provisioner)
	if err != nil {
		return err
	}
	var provisionerSlotsPerInstance int
	if p != nil {
		provisionerSlotsPerInstance = p.SlotsPerInstance()
	}

	m.proxy, _ = m.system.ActorOf(actor.Addr("proxy"), &proxy.Proxy{})

	m.initializeResourceProviders(provisionerSlotsPerInstance)

	m.system.ActorOf(actor.Addr("experiments"), &actors.Group{})

	rwCoordinator := newRWCoordinator()
	m.rwCoordinator, _ = m.system.ActorOf(actor.Addr("rwCoordinator"), rwCoordinator)

	// Find and restore all non-terminal experiments from the database.
	toRestore, err := m.db.NonTerminalExperiments()
	if err != nil {
		return errors.Wrap(err, "couldn't retrieve experiments to restore")
	}
	for _, exp := range toRestore {
		go m.restoreExperiment(exp)
	}

	// Used to decide whether we add trailing slash to the paths or not affecting
	// relative links in web pages hosted under these routes.
	staticWebDirectoryPaths := map[string]bool{
		"/swagger-ui": true,
		"/docs":       true,
	}

	// Initialize the HTTP server and listen for incoming requests.
	m.echo = echo.New()
	m.echo.Use(middleware.Recover())
	m.echo.Use(middleware.AddTrailingSlashWithConfig(middleware.TrailingSlashConfig{
		Skipper: func(c echo.Context) bool {
			return !staticWebDirectoryPaths[c.Path()]
		},
		RedirectCode: http.StatusMovedPermanently,
	}))

	// Add resistance to common HTTP attacks.
	//
	// TODO(DET-1696): Enable Content Security Policy (CSP).
	secureConfig := middleware.SecureConfig{
		Skipper:            middleware.DefaultSkipper,
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "SAMEORIGIN",
	}
	m.echo.Use(middleware.SecureWithConfig(secureConfig))

	// Register middleware that extends default context.
	m.echo.Use(func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &context.DetContext{Context: c}
			return h(cc)
		}
	})

	m.echo.Use(convertDBErrorsToNotFound)

	m.echo.Logger = logger.New()
	m.echo.HideBanner = true
	m.echo.HTTPErrorHandler = api.JSONErrorHandler

	webuiRoot := filepath.Join(m.config.Root, "webui")
	reactRoot := filepath.Join(webuiRoot, "react")

	// Docs.
	m.echo.Static("/docs", filepath.Join(webuiRoot, "docs"))

	type fileRoute struct {
		route string
		path  string
	}

	// React WebUI.
	reactIndexFiles := [...]fileRoute{
		{"/", "index.html"},
		{"/det", "index.html"},
		{"/det/*", "index.html"},
	}

	reactFiles := [...]fileRoute{
		{"/security.txt", "security.txt"},
		{"/.well-known/security.txt", "security.txt"},
		{"/color.less", "color.less"},
		{"/manifest.json", "manifest.json"},
		{"/favicon.ico", "favicon.ico"},
		{"/favicon.ico", "favicon.ico"},
	}

	reactDirs := [...]fileRoute{
		{"/favicons", "favicons"},
		{"/fonts", "fonts"},
		{"/static", "static"},
		{"/wait", "wait"},
	}

	for _, indexRoute := range reactIndexFiles {
		reactIndexPath := filepath.Join(reactRoot, indexRoute.path)
		m.echo.GET(indexRoute.route, func(c echo.Context) error {
			c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Response().Header().Set("Pragma", "no-cache")
			c.Response().Header().Set("Expires", "0")
			return c.File(reactIndexPath)
		})
	}

	for _, fileRoute := range reactFiles {
		m.echo.File(fileRoute.route, filepath.Join(reactRoot, fileRoute.path))
	}

	for _, dirRoute := range reactDirs {
		m.echo.Static(dirRoute.route, filepath.Join(reactRoot, dirRoute.path))
	}

	m.echo.Static("/swagger-ui", filepath.Join(m.config.Root, "static/swagger-ui"))
	m.echo.Static("/api/v1/api.swagger.json",
		filepath.Join(m.config.Root, "swagger/determined/api/v1/api.swagger.json"))

	m.echo.GET("/config", api.Route(m.getConfig))
	m.echo.GET("/info", api.Route(m.getInfo))
	m.echo.GET("/logs", api.Route(m.getMasterLogs), authFuncs...)

	m.echo.GET("/experiment-list", api.Route(m.getExperimentList), authFuncs...)
	m.echo.GET("/experiment-summaries", api.Route(m.getExperimentSummaries), authFuncs...)

	experimentsGroup := m.echo.Group("/experiments", authFuncs...)
	experimentsGroup.GET("", api.Route(m.getExperiments))
	experimentsGroup.GET("/:experiment_id", api.Route(m.getExperiment))
	experimentsGroup.GET("/:experiment_id/checkpoints", api.Route(m.getExperimentCheckpoints))
	experimentsGroup.GET("/:experiment_id/config", api.Route(m.getExperimentConfig))
	experimentsGroup.GET("/:experiment_id/model_def", m.getExperimentModelDefinition)
	experimentsGroup.GET("/:experiment_id/preview_gc", api.Route(m.getExperimentCheckpointsToGC))
	experimentsGroup.GET("/:experiment_id/summary", api.Route(m.getExperimentSummary))
	experimentsGroup.GET("/:experiment_id/metrics/summary", api.Route(m.getExperimentSummaryMetrics))
	experimentsGroup.PATCH("/:experiment_id", api.Route(m.patchExperiment))
	experimentsGroup.POST("", api.Route(m.postExperiment))
	experimentsGroup.POST("/:experiment_id/kill", api.Route(m.postExperimentKill))
	experimentsGroup.DELETE("/:experiment_id", api.Route(m.deleteExperiment))

	searcherGroup := m.echo.Group("/searcher", authFuncs...)
	searcherGroup.POST("/preview", api.Route(m.getSearcherPreview))

	tasksGroup := m.echo.Group("/tasks", authFuncs...)
	tasksGroup.GET("", api.Route(m.getTasks))
	tasksGroup.GET("/:task_id", api.Route(m.getTask))

	trialsGroup := m.echo.Group("/trials", authFuncs...)
	trialsGroup.GET("/:trial_id", api.Route(m.getTrial))
	trialsGroup.GET("/:trial_id/details", api.Route(m.getTrialDetails))
	trialsGroup.GET("/:trial_id/logs", m.getTrialLogs)
	trialsGroup.GET("/:trial_id/metrics", api.Route(m.getTrialMetrics))
	trialsGroup.GET("/:trial_id/logsv2", api.Route(m.getTrialLogsV2))
	trialsGroup.POST("/:trial_id/kill", api.Route(m.postTrialKill))

	checkpointsGroup := m.echo.Group("/checkpoints", authFuncs...)
	checkpointsGroup.GET("", api.Route(m.getCheckpoints))
	checkpointsGroup.GET("/:checkpoint_uuid", api.Route(m.getCheckpoint))
	checkpointsGroup.POST("/:checkpoint_uuid/metadata", api.Route(m.addCheckpointMetadata))
	checkpointsGroup.DELETE("/:checkpoint_uuid/metadata", api.Route(m.deleteCheckpointMetadata))

	m.echo.GET("/ws/trial/:experiment_id/:trial_id/:container_id",
		api.WebSocketRoute(m.trialWebSocket))

	m.echo.GET("/ws/data-layer/*",
		api.WebSocketRoute(m.rwCoordinatorWebSocket))

	m.echo.Any("/debug/pprof/*", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	m.echo.Any("/debug/pprof/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	m.echo.Any("/debug/pprof/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	m.echo.Any("/debug/pprof/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	m.echo.Any("/debug/pprof/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))

	handler := m.system.AskAt(actor.Addr("proxy"), proxy.NewProxyHandler{ServiceID: "service"})
	m.echo.Any("/proxy/:service/*", handler.Get().(echo.HandlerFunc))

	handler = m.system.AskAt(actor.Addr("proxy"), proxy.NewConnectHandler{})
	m.echo.CONNECT("*", handler.Get().(echo.HandlerFunc))

	user.RegisterAPIHandler(m.echo, userService, authFuncs...)
	command.RegisterAPIHandler(
		m.system,
		m.echo,
		m.db,
		m.proxy,
		m.config.TensorBoardTimeout,
		m.config.Security.DefaultTask,
		m.defaultTaskSpec,
		authFuncs...,
	)
	template.RegisterAPIHandler(m.echo, m.db, authFuncs...)

	// The Echo server registrations must be serialized, so we block until the ResourceProvider is
	// finished with its ConfigureEndpoints call.
	m.system.Ask(m.rp, sproto.ConfigureEndpoints{System: m.system, Echo: m.echo}).Get()

	if m.config.Telemetry.Enabled && m.config.Telemetry.SegmentMasterKey != "" {
		if telemetry, err := telemetry.NewActor(
			m.db,
			m.ClusterID,
			m.MasterID,
			m.Version,
			m.config.Telemetry.SegmentMasterKey,
		); err != nil {
			// We wouldn't want to totally fail just because telemetry failed; just note the error.
			log.WithError(err).Errorf("failed to initialize telemetry")
		} else {
			log.Info("telemetry reporting is enabled; run with `--telemetry-enabled=false` to disable")
			m.system.ActorOf(actor.Addr("telemetry"), telemetry)
		}
	} else {
		log.Info("telemetry reporting is disabled")
	}

	return m.startServers(cert)
}

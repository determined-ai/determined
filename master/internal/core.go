package internal

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/http/pprof"
	"net/url"
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

	"github.com/determined-ai/determined/master/internal/agent"
	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/internal/template"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
)

const defaultAskTimeout = 2 * time.Second

// Master manages the Determined master state.
type Master struct {
	ClusterID string
	MasterID  string
	Version   string

	config *Config

	logs          *logger.LogBuffer
	system        *actor.System
	echo          *echo.Echo
	cluster       *actor.Ref
	rwCoordinator *actor.Ref
	provisioner   *actor.Ref
	db            *db.PgDB
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

func (m *Master) hasuraMetaRequest(body []byte) error {
	const MaxRetries = 10
	const RetryDelay = 1

	client := &http.Client{}
	url := fmt.Sprintf("http://%s/v1/query", m.config.Hasura.Address)

	var err error

	for i := 0; i < MaxRetries; i++ {
		if i > 0 {
			time.Sleep(RetryDelay * time.Second)
		}

		// Construct the request.
		req, rErr := http.NewRequest("POST", url, bytes.NewReader(body))
		if rErr != nil {
			// This presumably won't change if we retry, so exit immediately.
			return rErr
		}
		req.Header.Add("X-Hasura-Admin-Secret", m.config.Hasura.Secret)
		req.Header.Add("Content-Type", "application/json")

		// In the code below, the uses of err refer to the variable declared outside the loop, so that we
		// can conveniently return its last value after exiting the loop.
		var resp *http.Response
		resp, err = client.Do(req)
		if err != nil {
			continue
		}

		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		if err = resp.Body.Close(); err != nil {
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			err = errors.Errorf("got error response from Hasura: %v %s", resp.Status, string(body))
			continue
		}

		return nil
	}
	return err
}

// updateHasuraSchema sends the current Hasura schema file to Hasura, ensuring that it is always up
// to date with the database schema.
func (m *Master) updateHasuraSchema() error {
	reloadBody, err := json.Marshal(map[string]interface{}{
		"type": "reload_metadata",
		"args": map[string]interface{}{},
	})
	if err != nil {
		return err
	}

	schemaBytes, err := ioutil.ReadFile(filepath.Join(m.config.Root, "static/hasura-metadata.json"))
	if err != nil {
		return err
	}

	schemaBody, err := json.Marshal(map[string]interface{}{
		"type": "replace_metadata",
		"args": json.RawMessage(schemaBytes),
	})
	if err != nil {
		return err
	}

	log.Info("telling Hasura to reload DB schema")
	if err = m.hasuraMetaRequest(reloadBody); err != nil {
		return errors.Wrap(err, "error updating Hasura DB metadata")
	}
	log.Info("updating Hasura GraphQL schema")
	if err = m.hasuraMetaRequest(schemaBody); err != nil {
		return errors.Wrap(err, "error updating Hasura schema")
	}
	log.Info("successfully updated Hasura metadata")
	return nil
}

func (m *Master) postGraphQL(c echo.Context) error {
	// The proxy appends the path of the current request's URL to the given base, which we don't want;
	// instead, pretend that this request was at "/", so it forwards to exactly the URL we provide.
	fakeURL, _ := url.Parse("/")
	c.Request().URL = fakeURL
	c.Request().Header.Add("X-Hasura-Admin-Secret", m.config.Hasura.Secret)
	c.Request().Header.Add("X-Hasura-Role", "user")

	graphqlURL, _ := url.Parse(fmt.Sprintf("http://%s/v1/graphql", m.config.Hasura.Address))
	proxy := httputil.NewSingleHostReverseProxy(graphqlURL)
	proxy.ServeHTTP(c.Response(), c.Request())

	return nil
}

func (m *Master) startServers() error {
	// Create the desired server configurations. The values of the servers map are descriptions to be
	// used in making error messages more informative.
	servers := make(map[*http.Server]string)

	if m.config.Security.HTTP {
		servers[&http.Server{
			Addr: fmt.Sprintf(":%d", m.config.HTTPPort),
		}] = "http server"
	}

	certFile := m.config.Security.TLS.Cert
	keyFile := m.config.Security.TLS.Key
	switch {
	case certFile == "" && keyFile != "":
		return errors.New("TLS key was provided without a cert")
	case certFile != "" && keyFile == "":
		return errors.New("TLS cert was provided without a key")
	case certFile != "" && keyFile != "":
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}

		servers[&http.Server{
			Addr: fmt.Sprintf(":%d", m.config.HTTPSPort),
			TLSConfig: &tls.Config{
				Certificates:             []tls.Certificate{cert},
				MinVersion:               tls.VersionTLS12,
				PreferServerCipherSuites: true,
			},
		}] = "https server"
	}

	if len(servers) == 0 {
		return errors.New("master was not configured to listen on any port")
	}

	// Start all servers.
	errs := make(chan error)
	defer close(errs)

	runServer := func(server *http.Server) {
		errs <- errors.Wrap(m.echo.StartServer(server), servers[server]+" failed")
	}
	for server := range servers {
		go runServer(server)
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

	go m.cleanUpSearcherEvents()

	// Actor structure:
	// master system
	// +- Provisioner (provisioner.Provisioner: provisioner)
	// +- Cluster (scheduler.Cluster: cluster)
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

	proxyRef, _ := m.system.ActorOf(actor.Addr("proxy"), &proxy.Proxy{})

	cluster := scheduler.NewCluster(
		m.ClusterID,
		m.config.Scheduler.MakeScheduler(),
		m.config.Scheduler.FitFunction(),
		proxyRef,
		filepath.Join(m.config.Root, "wheels"),
		m.config.TaskContainerDefaults,
		m.provisioner,
		provisionerSlotsPerInstance,
	)

	m.cluster, _ = m.system.ActorOf(actor.Addr("cluster"), cluster)
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

	// Initialize the HTTP server and listen for incoming requests.
	m.echo = echo.New()
	m.echo.Use(middleware.Recover())

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
	elmRoot := filepath.Join(webuiRoot, "elm")
	reactRoot := filepath.Join(webuiRoot, "react")

	// Docs.
	m.echo.Static("/docs", filepath.Join(webuiRoot, "docs"))
	m.echo.GET("/docs", func(c echo.Context) error {
		return c.Redirect(301, "/docs/")
	})

	type fileRoute struct {
		route string
		path  string
	}

	// Elm WebUI.
	elmFiles := [...]fileRoute{
		{"/ui", "public/index.html"},
		{"/ui/*", "public/index.html"},
		{"/wait", "public/wait.html"},
	}

	elmDirs := [...]fileRoute{
		{"/public", "public"},
	}

	// React WebUI.
	reactFiles := [...]fileRoute{
		{"/", "index.html"},
		{"/det", "index.html"},
		{"/security.txt", "security.txt"},
		{"/.well-known/security.txt", "security.txt"},
		{"/color.less", "color.less"},
		{"/manifest.json", "manifest.json"},
		{"/favicon.ico", "favicon.ico"},
		{"/favicon.ico", "favicon.ico"},
		{"/det/*", "index.html"},
	}

	reactDirs := [...]fileRoute{
		{"/favicons", "favicons"},
		{"/fonts", "fonts"},
		{"/static", "static"},
	}

	// Apply WebUI routes in order.
	for _, fileRoute := range elmFiles {
		m.echo.File(fileRoute.route, filepath.Join(elmRoot, fileRoute.path))
	}

	for _, dirRoute := range elmDirs {
		m.echo.Static(dirRoute.route, filepath.Join(elmRoot, dirRoute.path))
	}

	for _, fileRoute := range reactFiles {
		m.echo.File(fileRoute.route, filepath.Join(reactRoot, fileRoute.path))
	}

	for _, dirRoute := range reactDirs {
		m.echo.Static(dirRoute.route, filepath.Join(reactRoot, dirRoute.path))
	}

	m.echo.GET("/info", api.Route(m.getInfo))
	m.echo.GET("/logs", api.Route(m.getMasterLogs), authFuncs...)

	m.echo.POST("/graphql", m.postGraphQL, authFuncs...)

	m.echo.GET("/experiment-list", api.Route(m.getExperimentList), authFuncs...)

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
	tasksGroup.DELETE("/:task_id", api.Route(m.deleteTask))

	trialsGroup := m.echo.Group("/trials", authFuncs...)
	trialsGroup.GET("/:trial_id", api.Route(m.getTrial))
	trialsGroup.GET("/:trial_id/details", api.Route(m.getTrialDetails))
	trialsGroup.GET("/:trial_id/logs", m.getTrialLogs)
	trialsGroup.GET("/:trial_id/metrics", api.Route(m.getTrialMetrics))
	trialsGroup.GET("/:trial_id/logsv2", api.Route(m.getTrialLogsV2))
	trialsGroup.POST("/:trial_id/kill", api.Route(m.postTrialKill))

	m.echo.GET("/ws/trial/:experiment_id/:trial_id/:container_id",
		api.WebSocketRoute(m.trialWebSocket))

	m.echo.GET("/ws/data-layer/*",
		api.WebSocketRoute(m.rwCoordinatorWebSocket))

	m.echo.Any("/debug/pprof/*", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	m.echo.Any("/debug/pprof/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	m.echo.Any("/debug/pprof/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	m.echo.Any("/debug/pprof/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	m.echo.Any("/debug/pprof/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))

	handler := m.system.AskAt(actor.Addr("proxy"), proxy.NewHandler{ServiceKey: "service"})
	m.echo.Any("/proxy/:service/*", handler.Get().(echo.HandlerFunc))

	user.RegisterAPIHandler(m.echo, userService, authFuncs...)
	command.RegisterAPIHandler(
		m.system,
		m.echo,
		m.db,
		m.ClusterID,
		m.config.Security.DefaultTask,
		authFuncs...,
	)
	template.RegisterAPIHandler(m.echo, m.db, authFuncs...)

	agent.Initialize(m.system, m.echo, m.cluster)

	if err := m.updateHasuraSchema(); err != nil {
		return err
	}

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

	return m.startServers()
}

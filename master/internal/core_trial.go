package internal

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/actor"
	cproto "github.com/determined-ai/determined/master/pkg/container"
)

func (m *Master) postTrialKill(c echo.Context) (interface{}, error) {
	args := struct {
		TrialID int `path:"trial_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}

	trial, err := m.db.TrialByID(args.TrialID)
	if err != nil {
		return nil, err
	}
	resp := m.system.AskAt(actor.Addr("experiments", trial.ExperimentID),
		getTrial{trialID: args.TrialID})
	if resp.Source() == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("active experiment not found: %d", trial.ExperimentID))
	}
	if resp.Empty() {
		return nil, echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("active trial not found: %d", args.TrialID))
	}
	resp = m.system.AskAt(resp.Get().(*actor.Ref).Address(), killTrial{})
	if resp.Source() == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("active trial not found: %d", args.TrialID))
	}
	if _, notTimedOut := resp.GetOrTimeout(defaultAskTimeout); !notTimedOut {
		return nil, errors.Errorf("attempt to kill trial timed out")
	}
	return nil, nil
}

func (m *Master) getTrial(c echo.Context) (interface{}, error) {
	return m.db.RawQuery("get_trial", c.Param("trial_id"))
}

func (m *Master) getTrialDetails(c echo.Context) (interface{}, error) {
	args := struct {
		TrialID int `path:"trial_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	return m.db.TrialDetailsRaw(args.TrialID)
}

func (m *Master) getTrialMetrics(c echo.Context) (interface{}, error) {
	return m.db.RawQuery("get_trial_metrics", c.Param("trial_id"))
}

func (m *Master) trialWebSocket(socket *websocket.Conn, c echo.Context) error {
	args := struct {
		ExperimentID int    `path:"experiment_id"`
		TrialID      int    `path:"trial_id"`
		ContainerID  string `path:"container_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return err
	}

	c.Logger().Infof("new connection from container %v trial %d (experiment %d) at %v",
		args.ContainerID, args.TrialID, args.ExperimentID, socket.RemoteAddr())

	resp := m.system.AskAt(actor.Addr("experiments", args.ExperimentID),
		getTrial{trialID: args.TrialID})
	if resp.Source() == nil {
		return echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("active experiment not found: %d", args.ExperimentID))
	}
	if resp.Empty() {
		return echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("active trial not found: %d", args.TrialID))
	}

	// TODO: Better handling of sockets connecting to closing trials.

	// Notify the trial actor that a websocket is attempting to connect.
	socketActor := m.system.Ask(resp.Get().(*actor.Ref),
		containerConnected{ContainerID: cproto.ID(args.ContainerID), socket: socket})
	actorRef, ok := socketActor.Get().(*actor.Ref)
	if !ok {
		// TODO: Handle the case when multiple containers have been assigned to execute the same
		// trial.
		c.Logger().Infof("ignoring multiple connections from trial %d (experiment %d) at %v",
			args.TrialID, args.ExperimentID, socket.RemoteAddr())
		return nil
	}
	// Wait for the websocket actor to terminate.
	return actorRef.AwaitTermination()
}

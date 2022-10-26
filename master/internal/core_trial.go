package internal

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
)

func echoCanGetTrial(c echo.Context, m *Master, trialID string) error {
	id, err := strconv.Atoi(trialID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "trial ID must be numeric got %s", trialID)
	}

	curUser := c.(*detContext.DetContext).MustGetUser()
	trialNotFound := echo.NewHTTPError(http.StatusNotFound, "trial not found: %d", id)
	exp, err := m.db.ExperimentWithoutConfigByTrialID(id)
	if errors.Is(err, db.ErrNotFound) {
		return trialNotFound
	} else if err != nil {
		return err
	}
	var ok bool
	ctx := c.Request().Context()
	if ok, err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, curUser, exp); err != nil {
		return err
	} else if !ok {
		return trialNotFound
	}

	if err = expauth.AuthZProvider.Get().CanGetExperimentArtifacts(ctx, curUser, exp); err != nil {
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	}
	return nil
}

// TODO(ilia): These APIs are deprecated and will be removed in a future release.
func (m *Master) getTrial(c echo.Context) (interface{}, error) {
	if err := echoCanGetTrial(c, m, c.Param("trial_id")); err != nil {
		return nil, err
	}

	return m.db.RawQuery("get_trial", c.Param("trial_id"))
}

func (m *Master) getTrialMetrics(c echo.Context) (interface{}, error) {
	if err := echoCanGetTrial(c, m, c.Param("trial_id")); err != nil {
		return nil, err
	}

	return m.db.RawQuery("get_trial_metrics", c.Param("trial_id"))
}

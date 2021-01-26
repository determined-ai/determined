package internal

import (
	"io/ioutil"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

func (m *Master) getSearcherPreview(c echo.Context) (interface{}, error) {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	config, err := expconf.ParseAnyExperimentConfigYAML(body)
	if err != nil {
		return nil, errors.Wrap(err, "invalid experiment configuration")
	}

	schemas.FillDefaults(&config)

	err = schemas.IsComplete(&config.Searcher)
	if err != nil {
		return nil, errors.Wrap(err, "incomplete searcher config")
	}
	err = schemas.IsComplete(&config.Hyperparameters)
	if err != nil {
		return nil, errors.Wrap(err, "incomplete hyperparameters config")
	}

	sm := searcher.NewSearchMethod(config.Searcher)
	s := searcher.NewSearcher(0, sm, config.Hyperparameters)
	return searcher.Simulate(s, nil, searcher.RandomValidation, true, config.Searcher.Metric)
}

// cleanUpSearcherEvents deletes all searcher events for terminal state experiments from
// the database.
func (m *Master) cleanUpSearcherEvents() {
	log.Info("deleting all searcher events for terminal state experiments")
	err := m.db.DeleteSearcherEventsForTerminalStateExperiments()
	if err != nil {
		log.WithError(err).Errorf("cannot delete searcher events")
	}
}

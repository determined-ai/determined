package internal

import (
	"io/ioutil"

	"github.com/labstack/echo/v4"
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

	// Parse the provided experiment config.
	config, err := expconf.ParseAnyExperimentConfigYAML(body)
	if err != nil {
		return nil, errors.Wrap(err, "invalid experiment configuration")
	}

	// Apply any json-schema-defined defaults.
	config = schemas.WithDefaults(config).(expconf.ExperimentConfig)

	// Make sure the experiment config has all eventuallyRequired fields.
	err = schemas.IsComplete(config)
	if err != nil {
		return nil, errors.Wrap(err, "invalid experiment configuration")
	}

	sm := searcher.NewSearchMethod(config.Searcher())
	s := searcher.NewSearcher(0, sm, config.Hyperparameters())
	return searcher.Simulate(s, nil, searcher.RandomValidation, true, config.Searcher().Metric())
}

// cleanUpExperimentSnapshots deletes all snapshots for terminal state experiments from
// the database.
func (m *Master) cleanUpExperimentSnapshots() {
	log.Info("deleting all snapshots for terminal state experiments")
	if err := m.db.DeleteSnapshotsForTerminalExperiments(); err != nil {
		log.WithError(err).Errorf("cannot delete snapshots")
	}
}

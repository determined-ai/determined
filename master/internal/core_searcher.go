package internal

import (
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"

	"github.com/labstack/echo/v4"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

func (m *Master) getSearcherPreview(c echo.Context) (interface{}, error) {
	bytes, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	// Parse the provided experiment config.
	config, err := expconf.ParseAnyExperimentConfigYAML(bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid experiment configuration: %w", err)
	}

	// Get the useful subconfigs for preview search.
	if config.RawSearcher == nil {
		return nil, errors.New("invalid experiment configuration; missing searcher")
	}
	sc := *config.RawSearcher
	hc := config.RawHyperparameters

	// Apply any json-schema-defined defaults.
	sc = schemas.WithDefaults(sc)
	hc = schemas.WithDefaults(hc)

	// Make sure the searcher config has all eventuallyRequired fields.
	if err = schemas.IsComplete(sc); err != nil {
		return nil, fmt.Errorf("invalid searcher configuration: %w", err)
	}
	if err = schemas.IsComplete(hc); err != nil {
		return nil, fmt.Errorf("invalid hyperparameters configuration: %w", err)
	}

	// Disallow EOL searchers.
	if err = sc.AssertCurrent(); err != nil {
		return nil, fmt.Errorf("invalid experiment configuration: %w", err)
	}

	sm := searcher.NewSearchMethod(sc)
	s := searcher.NewSearcher(0, sm, hc)
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

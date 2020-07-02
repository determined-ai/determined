package internal

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/labstack/echo"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

func (m *Master) getSearcherPreview(c echo.Context) (interface{}, error) {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}
	config := model.DefaultExperimentConfig()
	if uerr := yaml.Unmarshal(body, &config); uerr != nil {
		return nil, uerr
	}
	if verr := check.Validate(config.Searcher); verr != nil {
		return nil, verr
	}

	sm := searcher.NewSearchMethod(config.Searcher.Shim(config.BatchesPerStep))
	s := searcher.NewSearcher(0, sm, config.Hyperparameters, config.BatchesPerStep, config.RecordsPerEpoch)
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

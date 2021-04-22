package internal

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/labstack/echo/v4"
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
	config := model.DefaultExperimentConfig(&m.config.TaskContainerDefaults)
	if uerr := yaml.Unmarshal(body, &config); uerr != nil {
		return nil, uerr
	}
	if verr := check.Validate(config.Searcher); verr != nil {
		return nil, verr
	}

	sm := searcher.NewSearchMethod(config.Searcher)
	s := searcher.NewSearcher(0, sm, config.Hyperparameters)
	return searcher.Simulate(s, nil, searcher.RandomValidation, true, config.Searcher.Metric)
}

// cleanUpExperimentSnapshots deletes all snapshots for terminal state experiments from
// the database.
func (m *Master) cleanUpExperimentSnapshots() {
	log.Info("deleting all snapshots for terminal state experiments")
	if err := m.db.DeleteSnapshotsForTerminalExperiments(); err != nil {
		log.WithError(err).Errorf("cannot delete snapshots")
	}
}

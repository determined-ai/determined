package internal

import (
	log "github.com/sirupsen/logrus"
)

// cleanUpExperimentSnapshots deletes all snapshots for terminal state experiments from
// the database.
func (m *Master) cleanUpExperimentSnapshots() {
	log.Info("deleting all snapshots for terminal state experiments")
	if err := m.db.DeleteSnapshotsForTerminalExperiments(); err != nil {
		log.WithError(err).Errorf("cannot delete snapshots")
	}
}

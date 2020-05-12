package internal

import (
	"encoding/json"

	"github.com/labstack/echo"
)

// ExportableCheckpoint is a checkpoint that can be downloaded via checkpoint export.
type ExportableCheckpoint struct {
	UUID              string          `db:"uuid" json:"uuid"`
	SmallerIsBetter   bool            `db:"smaller_is_better" json:"smaller_is_better"`
	Metric            string          `db:"metric" json:"metric"`
	CheckpointStorage json.RawMessage `db:"checkpoint_storage" json:"checkpoint_storage"`
	BatchNumber       int             `db:"batch_number" json:"batch_number"`
	StartTime         string          `db:"start_time" json:"start_time"`
	EndTime           string          `db:"end_time" json:"end_time"`
	Resources         json.RawMessage `db:"resources" json:"resources"`
	ValidationMetrics json.RawMessage `db:"metrics" json:"metrics"`
	ValidationState   string          `db:"validation_state" json:"validation_state"`
}

func (m *Master) getCheckpoint(c echo.Context) (interface{}, error) {
	checkpoint := ExportableCheckpoint{}
	err := m.db.Query("get_checkpoint", &checkpoint, c.Param("checkpoint_uuid"))
	return checkpoint, err
}

func (m *Master) getCheckpoints(c echo.Context) (interface{}, error) {
	var checkpoints []ExportableCheckpoint
	if eid := c.QueryParam("experiment_id"); eid != "" {
		if err := m.db.Query("get_checkpoints_for_experiment", &checkpoints, eid); err != nil {
			return nil, err
		}
	} else {
		tid := c.QueryParam("trial_id")
		if err := m.db.Query("get_checkpoints_for_trial", &checkpoints, tid); err != nil {
			return nil, err
		}
	}
	return checkpoints, nil
}

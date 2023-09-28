package ft

import (
	"context"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// we could probably implement just in memory for phase one.

// AddAlert persists an alert to the database.
func AddAlert(ctx context.Context, alert *model.TaskAlert) error {
	_, err := db.Bun().NewInsert().Model(alert).Exec(ctx)
	return err
}

// GetAlerts retrieves all alerts for a given job.
func GetAlerts(jobID string) (alerts []*model.TaskAlert, err error) {
	// has allocation id and task id, rp, node id
	// retrieve one of every action
	// active tasks only?

	// get all alerts with the given task id. group by action, merge node ids and device ids.
	/*
		SELECT task_id, action, array_agg(node_id) as node_ids, array_agg(device_ids) as device_ids
		FROM task_alerts
		WHERE task_id = 'task1'
		GROUP BY task_id, action;
	*/

	// generate some alerts:
	/*
		INSERT INTO task_alerts (task_id, node_id, device_ids, action)
		SELECT
		  CASE
			WHEN random() < 0.33 THEN 'task1'
			WHEN random() < 0.66 THEN 'task2'
			ELSE 'task3'
		  END,
		  'node' || floor(random()*100 + 1)::int,
		  jsonb_build_array(floor(random()*100 + 1)::int),
		  CASE
			WHEN random() < 0.5 THEN 'no_retry'
			ELSE 'hw_failure'
		  END
		FROM generate_series(1,100);

	*/
	return
}

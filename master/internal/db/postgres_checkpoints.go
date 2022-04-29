package db

import (
	"github.com/pkg/errors"
)

func (db *PgDB) GetDeleteCheckpointsInModelRegistry(deleteCheckpoints []string) (map[string]int, error) {
	var checkpointIDRows []struct {
		ID string
	}

	if err := db.queryRows(`
	SELECT DISTINCT(c.uuid::text) AS ID FROM checkpoints AS c 
	JOIN model_versions AS mv ON mv.checkpoint_uuid = c.uuid
	WHERE c.uuid::text IN (SELECT UNNEST($1::text[])); 
`, &checkpointIDRows, deleteCheckpoints); err != nil {
		return nil, errors.Wrap(err, "querying for all requested delete checkpoints registered in model registry")
	}

	checkpointIDs := make(map[string]int)

	for _, cRow := range checkpointIDRows {
		checkpointIDs[cRow.ID] = 1
	}

	return checkpointIDs, nil
}

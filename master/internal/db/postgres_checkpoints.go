package db

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

func (db *PgDB) GetDeleteCheckpointsInModelRegistry(deleteCheckpoints []string) ([]string, error) {
	// do I need to convert the above to deleteCheckpoints to a list uuids? What is the best way to do that? traversing and converting each element?

	var checkpointIDRows []struct {
		ID uuid.UUID
	}
	if err := db.queryRows(`
	SELECT uuid
    FROM checkpoints c JOIN 
	model_versions mv ON mv.checkpoint_uuid = c.uuid
	WHERE c.uuid IN unnest(deleteCheckpoints); 
`, &checkpointIDRows); err != nil {
		return nil, errors.Wrap(err, "querying for all checkpoints registered in model registry")
	}
	var checkpointIDs []string
	for _, r := range checkpointIDRows {
		checkpointIDs = append(checkpointIDs, r.ID.String())
	}
	return checkpointIDs, nil
}

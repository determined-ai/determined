package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// updateRunMetadata is a helper function that returns the closure to update the metadata of a run.
func updateRunMetadata(
	runID int,
	rawMetadata map[string]any,
	flatMetadata []model.RunMetadataIndex,
	result *map[string]any,
) func(context.Context, bun.Tx) error {
	return func(ctx context.Context, tx bun.Tx) error {
		var projectID int
		err := tx.NewSelect().Table("runs").
			Column("project_id").
			Where("id = ?", runID).
			For("UPDATE"). // pessimistically lock the run row.
			Scan(ctx, &projectID)
		if err != nil {
			return fmt.Errorf("querying run metadata: %w", err)
		}

		upsertResp := model.RunMetadata{
			RunID:    runID,
			Metadata: rawMetadata,
		}
		err = tx.NewInsert().
			Model(&upsertResp).
			On("CONFLICT (run_id) DO UPDATE").
			Set("metadata = EXCLUDED.metadata").
			Returning("metadata").
			Scan(ctx)
		if err != nil {
			return fmt.Errorf(
				"upserting run metadata on run(%d): %w", runID, err)
		}
		*result = upsertResp.Metadata

		// hydrate the flat metadata with relevant ids.
		for i := range flatMetadata {
			flatMetadata[i].RunID = runID
			flatMetadata[i].ProjectID = projectID
		}

		_, err = tx.NewDelete().Model(&model.RunMetadataIndex{}).Where("run_id = ?", runID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("deleting run metadata indexes for run(%d): %w", runID, err)
		}
		_, err = tx.NewInsert().Model(&flatMetadata).Exec(ctx)
		if err != nil {
			return fmt.Errorf("inserting run metadata indexes for run(%d): %w", runID, err)
		}
		return nil
	}
}

// UpdateRunMetadata updates the metadata of a run, including the metadata indexes.
func UpdateRunMetadata(
	ctx context.Context,
	runID int,
	rawMetadata map[string]any,
	flatMetadata []model.RunMetadataIndex,
) (result map[string]any, err error) {
	err = Bun().RunInTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelReadCommitted},
		updateRunMetadata(runID, rawMetadata, flatMetadata, &result),
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetRunMetadata returns the metadata of a run from the database.
// If the run does not have any metadata, it returns an empty map.
func GetRunMetadata(ctx context.Context, runID int) (map[string]any, error) {
	var metadata model.RunMetadata
	err := Bun().NewSelect().Model(&metadata).Where("run_id = ?", runID).Scan(ctx)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return map[string]any{}, nil
	case err != nil:
		return nil, err
	}
	return metadata.Metadata, nil
}

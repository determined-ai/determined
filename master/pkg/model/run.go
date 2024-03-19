package model

import (
	"github.com/uptrace/bun"
)

// RunMetadata represents the metadata associated with a run in the database.
type RunMetadata map[string]interface{}

// RunMetadataIndex is the bun model of a runMetadataIndex entry.
type RunMetadataIndex struct {
	bun.BaseModel `bun:"table:runs_metadata_index"`
	ID            int    `bun:"id,pk,autoincrement"`
	RunID         int    `bun:"run_id"`
	FlatKey       string `bun:"flat_key"`
	Value         string `bun:"value"`
	DataType      string `bun:"data_type"`
	ProjectID     int    `bun:"project_id"`
}

// Proto returns the proto representation of a runMetadata.
func (r RunMetadataIndex) Proto() *RunMetadataIndex {
	return &RunMetadataIndex{
		ID:        r.ID,
		RunID:     r.RunID,
		FlatKey:   r.FlatKey,
		Value:     r.Value,
		DataType:  r.DataType,
		ProjectID: r.ProjectID,
	}
}

package model_versions

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/uptrace/bun"
)

// ModelVersion represents a row from the `model_versions` table.
type ModelVersion struct {
	bun.BaseModel `bun:"table:model_versions"`

	ID              int           `bun:"id"`
	Version         int           `bun:"version"`
	CheckpointID    int           `bun:"checkpoint_id"`
	CreationTime    time.Time     `bun:"creation_time"`
	ModelID         int           `bun:"model_id"`
	Metadata        model.JSONObj `bun:"metadata"`
	Name            string        `bun:"name"`
	LastUpdatedTime time.Time     `bun:"last_updated_time"`
	Comment         string        `bun:"comment"`
	Notes           string        `bun:"readme"`
	Username        string        `bun:"username"`
}

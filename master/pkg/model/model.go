package model

import (
	"time"
)

// Model represents a row from the `models` table.
type Model struct {
	ID              int       `db:"id" json:"id"`
	Name            string    `db:"name" json:"name"`
	Description     string    `db:"description" json:"description"`
	Metadata        JSONObj   `db:"metadata" json:"metadata"`
	CreationTime    time.Time `db:"creation_time" json:"creation_time"`
	LastUpdatedTime time.Time `db:"last_updated_time" json:"last_updated_time"`
}

// ModelVersion represents a row from the `model_versions` table.
type ModelVersion struct {
	ID           int       `db:"id" json:"id"`
	Version      int       `db:"version" json:"version"`
	CheckpointID int       `db:"checkpoint_id" json:"checkpoint_id"`
	CreationTime time.Time `db:"creation_time" json:"creation_time"`
	ModelID      int       `db:"model_id" json:"model_id"`
	Metadata     JSONObj   `db:"metadata" json:"metadata"`
}

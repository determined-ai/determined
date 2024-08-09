package model

import (
	"github.com/uptrace/bun"
)

// RunMetadata is the bun model of a runMetadata entry.
type RunMetadata struct {
	bun.BaseModel `bun:"table:runs_metadata"`
	RunID         int
	Metadata      map[string]interface{}
}

// RunMetadataIndex is the bun model of a runMetadataIndex entry.
type RunMetadataIndex struct {
	bun.BaseModel  `bun:"table:runs_metadata_index"`
	ID             int      `bun:"id,pk,autoincrement"`
	RunID          int      `bun:"run_id"`
	FlatKey        string   `bun:"flat_key"`
	StringValue    *string  `bun:"string_value"`
	IntegerValue   *int     `bun:"integer_value"`
	FloatValue     *float64 `bun:"float_value"`
	BooleanValue   *bool    `bun:"boolean_value"`
	TimestampValue *string  `bun:"timestamp_value"`
	ProjectID      int      `bun:"project_id"`
	IsArrayElement bool     `bun:"is_array_element"`
}

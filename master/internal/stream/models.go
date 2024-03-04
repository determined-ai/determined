package stream

import (
	"time"

	"github.com/uptrace/bun"
)

const (
	// ModelsDeleteKey specifies the key for delete models.
	ModelsDeleteKey = "models_deleted"
	// ModelsUpsertKey specifies the key for upsert models.
	ModelsUpsertKey = "model"
	// modelChannel specifies the channel to listen to model events.
	modelChannel = "stream_model_chan"
)

// ModelMsg is a stream.Msg.
//
// determined:stream-gen source=server delete_msg=ModelsDeleted
type ModelMsg struct {
	bun.BaseModel `bun:"table:models"`

	// immutable attributes
	ID int `bun:"id,pk" json:"id"`

	// mutable attributes
	Name            string    `bun:"name" json:"name"`
	Description     string    `bun:"description" json:"description"`
	Archived        bool      `bun:"archived" json:"archived"`
	CreationTime    time.Time `bun:"creation_time" json:"creation_time"`
	Notes           JSONB     `bun:"notes" json:"notes"`
	WorkspaceID     int       `bun:"workspace_id" json:"workspace_id"`
	UserID          int       `bun:"user_id" json:"user_id"`
	LastUpdatedTime time.Time `bun:"last_updated_time" json:"last_updated_time"`
	Metadata        JSONB     `bun:"metadata" json:"metadata"`
	Labels          JSONB     `bun:"labels" json:"labels"`

	// metadata
	Seq int64 `bun:"seq" json:"seq"`
}

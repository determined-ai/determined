package model

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

// Model represents a row from the `models` table.
type Model struct {
	bun.BaseModel

	ID              int       `db:"id" json:"id"`
	Name            string    `db:"name" json:"name"`
	Description     string    `db:"description" json:"description"`
	Metadata        JSONObj   `db:"metadata" json:"metadata"`
	CreationTime    time.Time `db:"creation_time" json:"creation_time"`
	LastUpdatedTime time.Time `db:"last_updated_time" json:"last_updated_time"`
	Labels          []string  `db:"labels" json:"labels"`
	Archived        bool      `db:"archived" json:"archived"`
	NumVersions     int       `db:"num_versions" json:"num_versions"`
	Notes           string

	UserID UserID
	// "join users on models.user_id = users.id"
	Username string `bun:",rel:belongs-to,join:user_id=id`
}

func (m Model) ToProto(pc *protoutils.ProtoConverter) modelv1.Model {
	if pc.Error() != nil {
		return modelv1.Model{}
	}

	return modelv1.Model{
		Description:     m.Description,
		Metadata:        pc.ToStruct(m.Metadata, "metadata"),
		CreationTime:    pc.ToTimestamp(m.CreationTime),
		LastUpdatedTime: pc.ToTimestamp(m.LastUpdatedTime),
		Id:              pc.ToInt32(m.ID),
		NumVersions:     pc.ToInt32(m.NumVersions),
		Labels:          m.Labels,
		Username:        m.Username,
		Archived:        m.Archived,
		Notes:           m.Notes,
	}
}

// ModelVersion represents a row from the `model_versions` table.
type ModelVersion struct {
	bun.BaseModel

	ID              int       `db:"id" json:"id"`
	Version         int       `db:"version" json:"version"`
	CheckpointID    int       `db:"checkpoint_id" json:"checkpoint_id"`
	CreationTime    time.Time `db:"creation_time" json:"creation_time"`
	ModelID         int       `db:"model_id" json:"model_id"`
	Metadata        JSONObj   `db:"metadata" json:"metadata"`
	Name            string    `db:"name" json:"name"`
	LastUpdatedTime time.Time `db:"last_updated_time" json:"last_updated_time"`
	Comment         string    `db:"comment" json:"comment"`
	Notes           string    `db:"notes" json:"notes"` // XXX: why was this db:"readme"?
	Username        string    `db:"username" json:"username"`
	Labels          Labels

	UserID UserID
	// "join users on models.user_id = users.id"
	// XXX: does this get picked up automatically??
	Username string `bun:",rel:belongs-to,join:user_id=id`
}

func (v ModelVersion) ToProto(
	pc *protoutils.ProtoConverter, model Model, checkpoint CheckpointExpanded,
) modelv1.ModelVersion {
	if pc.Error() != nil {
		return modelv1.ModelVersion{}
	}

	m := model.ToProto(pc)
	c := checkpoint.ToProto(pc)

	return modelv1.ModelVersion{
		Model:           &m,
		Checkpoint:      &c,
		Version:         pc.ToInt32(v.Version),
		CreationTime:    pc.ToTimestamp(v.CreationTime),
		Id:              pc.ToInt32(v.ID),
		Name:            v.Name,
		Metadata:        pc.ToStruct(v.Metadata, "metadata"),
		LastUpdatedTime: pc.ToTimestamp(v.LastUpdatedTime),
		Comment:         v.Comment, // XXX: why do we have .Comment and .Notes?
		Notes:           v.Notes,
		Username:        v.Username,
		Labels:          v.Labels,
	}
}

// Labels maps filenames to file sizes.
type Labels []string

// Scan converts jsonb from postgres into a Resources object.
// TODO: Combine all json.unmarshal-based Scanners into a single Scan implementation.
func (l *Labels) Scan(src interface{}) error {
	if src == nil {
		*l = nil
		return nil
	}
	bytes, ok := src.([]byte)
	if !ok {
		return errors.Errorf("unable to convert to []byte: %v", src)
	}
	obj := []string{}
	if err := json.Unmarshal(bytes, &obj); err != nil {
		return errors.Wrapf(err, "unable to unmarshal Labels: %v", src)
	}
	*l = Labels(obj)
	return nil
}

package model

import (
	"encoding/json"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/proto/pkg/modelv1"
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

// ModelToProto converts a model.Model to a modelv1.Model.
func ModelToProto(m Model) (*modelv1.Model, error) {
	configStruct := &structpb.Struct{}
	b, err := json.Marshal(m.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling model.Metadata")
	}
	err = protojson.Unmarshal(b, configStruct)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshaling metadata to protobuf struct")
	}
	creationTime, _ := ptypes.TimestampProto(m.CreationTime)
	lastUpdatedTime, _ := ptypes.TimestampProto(m.LastUpdatedTime)
	return &modelv1.Model{
			Name:            m.Name,
			Metadata:        configStruct,
			CreationTime:    creationTime,
			LastUpdatedTime: lastUpdatedTime,
		},
		nil
}

// ModelFromProto converts a modelv1.Model to a model.Model.
func ModelFromProto(model *modelv1.Model) (*Model, error) {
	b, err := protojson.Marshal(model.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling model.Metadata")
	}

	metadata := JSONObj{}
	err = json.Unmarshal(b, &metadata)
	if err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal metadata to model.Metadata")
	}

	creationTime, err := ptypes.Timestamp(model.CreationTime)
	if err != nil {
		return nil, errors.Wrap(err, "modelv1.CreationTime is not parsable to time.Time")
	}

	lastUpdatedTime, err := ptypes.Timestamp(model.LastUpdatedTime)
	if err != nil {
		return nil, errors.Wrap(err, "modelv1.LastUpdatedTime is not parsable to time.Time")
	}

	return &Model{Name: model.Name,
		Description:     model.Description,
		Metadata:        metadata,
		CreationTime:    creationTime,
		LastUpdatedTime: lastUpdatedTime,
	}, nil
}

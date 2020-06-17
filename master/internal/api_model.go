package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

// ModelToProto converts a model.Model to a modelv1.Model.
func ModelToProto(m model.Model) (*modelv1.Model, error) {
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
func ModelFromProto(m *modelv1.Model) (*model.Model, error) {
	b, err := protojson.Marshal(m.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling model.Metadata")
	}

	metadata := model.JSONObj{}
	err = json.Unmarshal(b, &metadata)
	if err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal metadata to model.Metadata")
	}

	creationTime, err := ptypes.Timestamp(m.CreationTime)
	if err != nil {
		return nil, errors.Wrap(err, "modelv1.CreationTime is not parsable to time.Time")
	}

	lastUpdatedTime, err := ptypes.Timestamp(m.LastUpdatedTime)
	if err != nil {
		return nil, errors.Wrap(err, "modelv1.LastUpdatedTime is not parsable to time.Time")
	}

	return &model.Model{Name: m.Name,
		Description:     m.Description,
		Metadata:        metadata,
		CreationTime:    creationTime,
		LastUpdatedTime: lastUpdatedTime,
	}, nil
}

func (a *apiServer) GetModel(
	_ context.Context, req *apiv1.GetModelRequest) (*apiv1.GetModelResponse, error) {
	switch m, err := a.m.db.ModelByName(req.ModelName); err {
	case nil:
		protoTemp, pErr := ModelToProto(m)
		return &apiv1.GetModelResponse{Model: protoTemp}, pErr
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "model %s not found", req.ModelName)
	default:
		return nil, errors.Wrapf(err, "error fetching model %s from database", req.ModelName)
	}
}

func (a *apiServer) PutModel(_ context.Context, req *apiv1.PutModelRequest) (*apiv1.PutModelResponse, error) {
	m, err := model.ModelFromProto(req.GetModel())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "model could not be parsed")
	}

	err = a.m.db.AddModel(m)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to add model %s", req.GetModel().GetName())
	}

	protoModel, err := model.ModelToProto(*m)
	return &apiv1.PutModelResponse{Model: protoModel}, err
}

func (a *apiServer) GetModels(_ context.Context, req *apiv1.GetModelsRequest) (*apiv1.GetModelsResponse, error) {
	fmt.Printf("req = %+v\n", req)
	switch m, err := a.m.db.ModelList(); err {
	case nil:
		models := []*modelv1.Model{}
		for _, v := range m {
			protoTemp, pErr := model.ModelToProto(v)
			if err != nil {
				return nil, pErr
			}

			models = append(models, protoTemp)
		}

		return &apiv1.GetModelsResponse{Models: models}, nil
	default:
		return nil, errors.Wrap(err, "error fetching models from database")
	}
}

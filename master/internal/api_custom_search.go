package internal

import (
	"context"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetSearcherEvents(_ context.Context, _ *apiv1.GetSearcherEventsRequest) (*apiv1.GetSearcherEventsResponse, error) {
	return &apiv1.GetSearcherEventsResponse{}, nil
}

func (a *apiServer) PostSearcherOperations(_ context.Context, _ *apiv1.PostSearcherOperationsRequest) (*apiv1.PostSearcherOperationsResponse, error) {
	return &apiv1.PostSearcherOperationsResponse{}, nil
}

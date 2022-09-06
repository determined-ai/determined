//go:build integration
// +build integration

package internal

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func errAllocationNotFound(id string) error {
	return status.Errorf(codes.NotFound, "allocation not found: %s", id)
}

func createTestAllocation(
	t *testing.T, api *apiServer, curUser model.User,
) *model.Allocation {
	trial := createTestTrial(t, api, curUser)
	alloc := &model.Allocation{
		AllocationID: model.AllocationID(uuid.New().String()),
		TaskID:       trial.TaskID,
		Slots:        1,
		AgentLabel:   "label",
		ResourcePool: "kubernetes",
	}
	require.NoError(t, api.m.db.AddAllocation(alloc))

	return alloc
}

func TestAllocationAuthZ(t *testing.T) {
	api, authZExp, _, curUser, ctx := SetupExpAuthTest(t)
	alloc := createTestAllocation(t, api, curUser)
	allocID := string(alloc.AllocationID)

	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id string) error
	}{
		{"CanEditExperiment", func(id string) error {
			_, err := api.AllocationReady(ctx, &apiv1.AllocationReadyRequest{
				AllocationId: id,
			})
			return err
		}},
		{"CanEditExperiment", func(id string) error {
			_, err := api.AllocationAllGather(ctx, &apiv1.AllocationAllGatherRequest{
				AllocationId: id,
			})
			return err
		}},
		{"CanEditExperiment", func(id string) error {
			_, err := api.PostAllocationProxyAddress(ctx, &apiv1.PostAllocationProxyAddressRequest{
				AllocationId: id,
			})
			return err
		}},
	}

	for _, curCase := range cases {
		require.Equal(t, curCase.IDToReqCall("-999"), errAllocationNotFound("-999"))

		// Can't view allocation's experiment gives same error.
		authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(false, nil).Once()
		require.Equal(t, curCase.IDToReqCall(allocID), errAllocationNotFound(allocID))

		// Experiment view error is returned unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(false, expectedErr).Once()
		require.Equal(t, curCase.IDToReqCall(allocID), expectedErr)

		// Action func error returns err in forbidden.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(true, nil).Once()
		authZExp.On(curCase.DenyFuncName, curUser, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.Equal(t, curCase.IDToReqCall(allocID), expectedErr)
	}
}

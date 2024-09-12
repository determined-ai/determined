package internal

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/determined-ai/determined/master/internal/configpolicy"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeleteWorkspaceConfigPolicies(t *testing.T) {
	// TODO (CM-520): Make test cases for experiment config policies.

	// Create one workspace and continuously set and delete config policies from there
	api, _, ctx := setupAPITest(t, nil)
	wkspResp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	workspaceID := wkspResp.Workspace.Id
	cases := []struct {
		name string
		req  *apiv1.DeleteWorkspaceConfigPoliciesRequest
		err  error
	}{
		{"invalid workload type",
			&apiv1.DeleteWorkspaceConfigPoliciesRequest{
				WorkspaceId:  workspaceID,
				WorkloadType: "bad workload type",
			},
			fmt.Errorf("invalid workload type"),
		},
		{
			"valid request",
			&apiv1.DeleteWorkspaceConfigPoliciesRequest{
				WorkspaceId:  workspaceID,
				WorkloadType: model.NTSCType.String(),
			},
			nil,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ntscPolicies := &model.NTSCTaskConfigPolicies{
				WorkspaceID:  ptrs.Ptr(int(test.req.WorkspaceId)),
				WorkloadType: model.NTSCType,
			}
			configpolicy.SetNTSCConfigPolicies(ctx, ntscPolicies)

			resp, err := api.DeleteWorkspaceConfigPolicies(ctx, test.req)
			if test.err != nil {
				require.ErrorContains(t, err, test.err.Error())
				return
			}
			// Delete successful?
			require.Nil(t, err)
			require.NotNil(t, resp)

			// Policies removed?
			policies, err := configpolicy.GetNTSCConfigPolicies(ctx, ptrs.Ptr(int(workspaceID)))
			require.Nil(t, policies)
			require.ErrorIs(t, err, sql.ErrNoRows)
		})
	}

	// Test invalid workspace ID.
	resp, err := api.DeleteWorkspaceConfigPolicies(ctx, &apiv1.DeleteWorkspaceConfigPoliciesRequest{
		WorkspaceId:  -1,
		WorkloadType: model.NTSCType.String(),
	})
	require.Nil(t, resp)
	require.ErrorContains(t, err, "InvalidArgument")
}

func TestDeleteGlobalConfigPolicies(t *testing.T) {
	// TODO (CM-520): Make test cases for experiment config policies.

	api, _, ctx := setupAPITest(t, nil)
	cases := []struct {
		name string
		req  *apiv1.DeleteGlobalConfigPoliciesRequest
		err  error
	}{
		{"invalid workload type",
			&apiv1.DeleteGlobalConfigPoliciesRequest{
				WorkloadType: "bad workload type",
			},
			fmt.Errorf("invalid workload type"),
		},
		{
			"valid request",
			&apiv1.DeleteGlobalConfigPoliciesRequest{
				WorkloadType: model.NTSCType.String(),
			},
			nil,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			configpolicy.SetNTSCConfigPolicies(ctx, &model.NTSCTaskConfigPolicies{
				WorkloadType: model.NTSCType,
			})

			resp, err := api.DeleteGlobalConfigPolicies(ctx, test.req)
			if test.err != nil {
				require.ErrorContains(t, err, test.err.Error())
				return
			}
			// Delete successful?
			require.Nil(t, err)
			require.NotNil(t, resp)

			// Policies removed?
			policies, err := configpolicy.GetNTSCConfigPolicies(ctx, nil)
			require.Nil(t, policies)
			require.ErrorIs(t, err, sql.ErrNoRows)
		})
	}
}

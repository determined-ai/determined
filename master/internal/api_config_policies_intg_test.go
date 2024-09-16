package internal

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/configpolicy"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/test/testutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestDeleteWorkspaceConfigPolicies(t *testing.T) {
	// TODO (CM-520): Make test cases for experiment config policies.

	// Create one workspace and continuously set and delete config policies from there
	api, _, ctx := setupAPITest(t, nil)
	testutils.MustLoadLicenseAndKeyFromFilesystem("../../")

	wkspResp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	workspaceID := wkspResp.Workspace.Id
	cases := []struct {
		name string
		req  *apiv1.DeleteWorkspaceConfigPoliciesRequest
		err  error
	}{
		{
			"invalid workload type",
			&apiv1.DeleteWorkspaceConfigPoliciesRequest{
				WorkspaceId:  workspaceID,
				WorkloadType: "bad workload type",
			},
			fmt.Errorf("invalid workload type"),
		},
		{
			"empty workload type",
			&apiv1.DeleteWorkspaceConfigPoliciesRequest{
				WorkspaceId:  workspaceID,
				WorkloadType: "no workload type",
			},
			fmt.Errorf("no workload type"),
		},
		{
			"valid request",
			&apiv1.DeleteWorkspaceConfigPoliciesRequest{
				WorkspaceId:  workspaceID,
				WorkloadType: model.NTSCType,
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
			err = configpolicy.SetNTSCConfigPolicies(ctx, ntscPolicies)
			require.NoError(t, err)

			resp, err := api.DeleteWorkspaceConfigPolicies(ctx, test.req)
			if test.err != nil {
				require.ErrorContains(t, err, test.err.Error())
				return
			}
			// Delete successful?
			require.NoError(t, err)
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
		WorkloadType: model.NTSCType,
	})
	require.Nil(t, resp)
	require.ErrorContains(t, err, "InvalidArgument")
}

func TestDeleteGlobalConfigPolicies(t *testing.T) {
	// TODO (CM-520): Make test cases for experiment config policies.

	api, _, ctx := setupAPITest(t, nil)
	testutils.MustLoadLicenseAndKeyFromFilesystem("../../")

	cases := []struct {
		name string
		req  *apiv1.DeleteGlobalConfigPoliciesRequest
		err  error
	}{
		{
			"invalid workload type",
			&apiv1.DeleteGlobalConfigPoliciesRequest{
				WorkloadType: "invalid workload type",
			},
			fmt.Errorf("invalid workload type"),
		},
		{
			"empty workload type",
			&apiv1.DeleteGlobalConfigPoliciesRequest{
				WorkloadType: "no workload type",
			},
			fmt.Errorf("no workload type"),
		},
		{
			"valid request",
			&apiv1.DeleteGlobalConfigPoliciesRequest{
				WorkloadType: model.NTSCType,
			},
			nil,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := configpolicy.SetNTSCConfigPolicies(ctx, &model.NTSCTaskConfigPolicies{
				WorkloadType: model.NTSCType,
			})
			require.NoError(t, err)

			resp, err := api.DeleteGlobalConfigPolicies(ctx, test.req)
			if test.err != nil {
				require.ErrorContains(t, err, test.err.Error())
				return
			}
			// Delete successful?
			require.NoError(t, err)
			require.NotNil(t, resp)

			// Policies removed?
			policies, err := configpolicy.GetNTSCConfigPolicies(ctx, nil)
			require.Nil(t, policies)
			require.ErrorIs(t, err, sql.ErrNoRows)
		})
	}
}

func TestBasicRBACConfigPolicyPerms(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	curUser.Admin = false
	err := user.Update(ctx, &curUser, []string{"admin"}, nil)
	require.NoError(t, err)

	testutils.MustLoadLicenseAndKeyFromFilesystem("../../")

	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	wkspID := resp.Workspace.Id

	wksp, err := workspace.WorkspaceByName(ctx, resp.Workspace.Name)
	require.NoError(t, err)
	newUser, err := db.HackAddUser(ctx, &model.User{Username: uuid.NewString()})
	require.NoError(t, err)

	wksp.UserID = newUser
	_, err = db.Bun().NewUpdate().Model(wksp).Where("id = ?", wksp.ID).Exec(ctx)
	require.NoError(t, err)

	cases := []struct {
		name string
		req  func() error
		err  error
	}{
		{
			"delete workspace config policies",
			func() error {
				_, err := api.DeleteWorkspaceConfigPolicies(ctx,
					&apiv1.DeleteWorkspaceConfigPoliciesRequest{
						WorkspaceId:  wkspID,
						WorkloadType: model.NTSCType,
					},
				)
				return err
			},
			fmt.Errorf("only admins may set config policies for workspaces"),
		},
		{
			"delete workspace config policies",
			func() error {
				_, err := api.DeleteGlobalConfigPolicies(ctx,
					&apiv1.DeleteGlobalConfigPoliciesRequest{
						WorkloadType: model.NTSCType,
					},
				)
				return err
			},
			fmt.Errorf("PermissionDenied"),
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := test.req()
			require.ErrorContains(t, err, test.err.Error())
		})
	}
}

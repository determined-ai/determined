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
	api, curUser, ctx := setupAPITest(t, nil)
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
				WorkloadType: "",
			},
			fmt.Errorf(noWorkloadErr),
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
			ntscPolicies := &model.TaskConfigPolicies{
				WorkspaceID:   ptrs.Ptr(int(test.req.WorkspaceId)),
				WorkloadType:  model.NTSCType,
				LastUpdatedBy: curUser.ID,
			}
			err = configpolicy.SetTaskConfigPolicies(ctx, ntscPolicies)
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
			policies, err := configpolicy.GetTaskConfigPolicies(ctx, ptrs.Ptr(int(workspaceID)), test.req.WorkloadType)
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
	require.ErrorContains(t, err, "not found")
}

func TestDeleteGlobalConfigPolicies(t *testing.T) {
	// TODO (CM-520): Make test cases for experiment config policies.

	api, curUser, ctx := setupAPITest(t, nil)
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
				WorkloadType: "",
			},
			fmt.Errorf(noWorkloadErr),
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
			err := configpolicy.SetTaskConfigPolicies(ctx, &model.TaskConfigPolicies{
				WorkloadType:  model.NTSCType,
				LastUpdatedBy: curUser.ID,
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
			policies, err := configpolicy.GetTaskConfigPolicies(ctx, nil, test.req.WorkloadType)
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
			"delete global config policies",
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

func TestGetConfigPolicies(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	testutils.MustLoadLicenseAndKeyFromFilesystem("../../")

	wkspResp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	workspaceID1 := ptrs.Ptr(int(wkspResp.Workspace.Id))
	wkspResp, err = api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	workspaceID2 := ptrs.Ptr(int(wkspResp.Workspace.Id))

	// set only constraints policy for workspace 1
	taskConfigPolicies := &model.TaskConfigPolicies{
		WorkspaceID:   workspaceID1,
		WorkloadType:  model.ExperimentType,
		LastUpdatedBy: curUser.ID,
		Constraints:   ptrs.Ptr(configpolicy.DefaultConstraintsStr),
	}
	err = configpolicy.SetTaskConfigPolicies(ctx, taskConfigPolicies)
	require.NoError(t, err)

	// set only config policy for workspace 1
	taskConfigPolicies.WorkloadType = model.NTSCType
	taskConfigPolicies.Constraints = nil
	taskConfigPolicies.InvariantConfig = ptrs.Ptr(configpolicy.DefaultInvariantConfigStr)
	err = configpolicy.SetTaskConfigPolicies(ctx, taskConfigPolicies)
	require.NoError(t, err)

	// set both config and constraints policy for workspace 2
	taskConfigPolicies.WorkspaceID = workspaceID2
	taskConfigPolicies.Constraints = ptrs.Ptr(configpolicy.DefaultConstraintsStr)
	err = configpolicy.SetTaskConfigPolicies(ctx, taskConfigPolicies)
	require.NoError(t, err)

	// set both config and constraints policy globally
	taskConfigPolicies.WorkspaceID = nil
	err = configpolicy.SetTaskConfigPolicies(ctx, taskConfigPolicies)
	require.NoError(t, err)

	cases := []struct {
		name           string
		workspaceID    *int
		workloadType   string
		err            error
		hasConfig      bool
		hasConstraints bool
	}{
		{
			"invalid workload type",
			workspaceID1,
			"bad workload type",
			fmt.Errorf("invalid workload type"),
			false,
			false,
		},
		{
			"empty workload type",
			workspaceID1,
			"",
			fmt.Errorf(noWorkloadErr),
			false,
			false,
		},
		{
			"valid workspace request, only config",
			workspaceID1,
			model.NTSCType,
			nil,
			true,
			false,
		},
		{
			"valid workspace request, only constraints",
			workspaceID1,
			model.ExperimentType,
			nil,
			false,
			true,
		},
		{
			"valid workspace request both configs and constraints",
			workspaceID2,
			model.NTSCType,
			nil,
			true,
			true,
		},
		{
			"valid workspace request, only config",
			workspaceID1,
			model.NTSCType,
			nil,
			true,
			false,
		},
		{
			"valid workspace request, only constraints",
			workspaceID1,
			model.ExperimentType,
			nil,
			false,
			true,
		},
		{
			"valid global request both configs and constraints",
			nil,
			model.NTSCType,
			nil,
			true,
			true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			resp, err := api.GetConfigPolicies(ctx, test.workspaceID, test.workloadType)
			if test.err != nil {
				require.ErrorContains(t, err, test.err.Error())
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)

			if test.hasConfig {
				require.Contains(t, resp.String(), "config")
			} else {
				require.NotContains(t, resp.String(), "config")
			}

			if test.hasConstraints {
				require.Contains(t, resp.String(), "constraints")
			} else {
				require.NotContains(t, resp.String(), "constraints")
			}
		})
	}
}

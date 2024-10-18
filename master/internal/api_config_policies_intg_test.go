//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/configpolicy"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/test/testutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	validConstraintsPolicyYAML = `
constraints:
  resources:
    max_slots: 4
  priority_limit: 10
`

	validExperimentConfigPolicyYAML = `
invariant_config:
  description: "test\nspecial\tchar"
  environment:
    force_pull_image: false
    add_capabilities:
      - "cap1"
      - "cap2"
  resources:
    slots: 1
  name: my_experiment_config
`

	validNTSCConfigPolicyYAML = `
invariant_config:
  description: "test\nspecial\tchar"
  environment:
    force_pull_image: false
    add_capabilities:
      - "cap1"
      - "cap2"
  resources:
    slots: 1
  work_dir: my/working/directory
`

	validConstraintsPolicyJSON = `
 	"constraints": {
        "resources": {
            "max_slots": 4
        },
        "priority_limit": 10
    }`

	validExperimentConfigPolicyJSON = `
	"invariant_config": {
        "description": "test\nspecial\tchar",
        "environment": {
            "force_pull_image": false,
            "add_capabilities": ["cap1", "cap2"]
        },
        "resources": {
            "slots": 1
        },
		"name": "my_experiment_config"
    }`
	validNTSCConfigPolicyJSON = `
	"invariant_config": {
		"description": "test\nspecial\tchar",
		"environment": {
			"force_pull_image": false,
			"add_capabilities": ["cap1", "cap2"]
		},
		"resources": {
			"slots": 1
		},
		"work_dir": "my/working/directory"
	}`

	invalidExperimentConfigPolicyErr = "invalid experiment config policy"
	invalidNTSCtConfigPolicyErr      = "invalid NTSC config policy"

	updatedExperimentConfigPolicyJSON = `
	"invariant_config": {
        "description": "test\nspecial\tchar",
        "environment": {
            "force_pull_image": false,
            "add_capabilities": ["cap1", "cap2"]
        },
        "resources": {
            "slots": 1
        },
		"name": "my_experiment_config",
		"entrypoint": "start from here"
    }
`

	entrypointYAML = `
  entrypoint: "start from here"
`
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
			require.NoError(t, err)
			require.Nil(t, policies.InvariantConfig)
			require.Nil(t, policies.Constraints)
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
			require.NoError(t, err)
			require.Nil(t, policies.InvariantConfig)
			require.Nil(t, policies.Constraints)
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
		{
			"put workspace config policies",
			func() error {
				_, err := api.PutWorkspaceConfigPolicies(ctx,
					&apiv1.PutWorkspaceConfigPoliciesRequest{
						WorkspaceId:    wkspID,
						WorkloadType:   model.NTSCType,
						ConfigPolicies: validNTSCConfigPolicyYAML + validConstraintsPolicyYAML,
					},
				)
				return err
			},
			fmt.Errorf("only admins may set config policies for workspaces"),
		},
		{
			"put global config policies",
			func() error {
				_, err := api.PutGlobalConfigPolicies(ctx,
					&apiv1.PutGlobalConfigPoliciesRequest{
						WorkloadType:   model.NTSCType,
						ConfigPolicies: validNTSCConfigPolicyYAML + validConstraintsPolicyYAML,
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

	setUpTaskConfigPolicies(ctx, t, workspaceID1, workspaceID2, curUser.ID)

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
			"valid global request both configs and constraints",
			nil,
			model.NTSCType,
			nil,
			true,
			true,
		},
		{
			"no global config policy",
			nil,
			model.ExperimentType,
			nil,
			false,
			false,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			resp, err := api.getConfigPolicies(ctx, test.workspaceID, test.workloadType)
			if test.err != nil {
				require.ErrorContains(t, err, test.err.Error())
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)

			if test.hasConfig {
				require.Contains(t, resp.String(), "invariant_config")
			} else {
				require.NotContains(t, resp.String(), "invariant_config")
			}

			if test.hasConstraints {
				require.Contains(t, resp.String(), "constraints")
			} else {
				require.NotContains(t, resp.String(), "constraints")
			}
		})
	}
}

func setUpTaskConfigPolicies(ctx context.Context, t *testing.T,
	workspaceID1 *int, workspaceID2 *int, userID model.UserID,
) {
	// set only Experiment constraints policy for workspace 1
	taskConfigPolicies := &model.TaskConfigPolicies{
		WorkspaceID:   workspaceID1,
		WorkloadType:  model.ExperimentType,
		LastUpdatedBy: userID,
		Constraints:   ptrs.Ptr(configpolicy.DefaultConstraintsStr),
	}
	err := configpolicy.SetTaskConfigPolicies(ctx, taskConfigPolicies)
	require.NoError(t, err)

	// set only NTSC config policy for workspace 1
	taskConfigPolicies.WorkloadType = model.NTSCType
	taskConfigPolicies.Constraints = nil
	taskConfigPolicies.InvariantConfig = ptrs.Ptr(configpolicy.DefaultInvariantConfigStr)
	err = configpolicy.SetTaskConfigPolicies(ctx, taskConfigPolicies)
	require.NoError(t, err)

	// set both config and constraints policy for workspace 2 (NTSC)
	taskConfigPolicies.WorkspaceID = workspaceID2
	taskConfigPolicies.Constraints = ptrs.Ptr(configpolicy.DefaultConstraintsStr)
	err = configpolicy.SetTaskConfigPolicies(ctx, taskConfigPolicies)
	require.NoError(t, err)

	// set both config and constraints policy globally (NTSC)
	taskConfigPolicies.WorkspaceID = nil
	err = configpolicy.SetTaskConfigPolicies(ctx, taskConfigPolicies)
	require.NoError(t, err)
}

func TestAuthZCanModifyConfigPolicies(t *testing.T) {
	api, workspaceAuthZ, _, ctx := setupWorkspaceAuthZTest(t, nil)
	testutils.MustLoadLicenseAndKeyFromFilesystem("../../")
	configPolicyAuthZ := setupConfigPolicyAuthZ()

	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).Return(nil).Once()

	wkspResp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	workspaceID := wkspResp.Workspace.Id

	// (Workspace-level) Deny with permission access error.
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Twice()
	expectedErr := fmt.Errorf("canModifyConfigPoliciesError")
	configPolicyAuthZ.On("CanModifyWorkspaceConfigPolicies", mock.Anything, mock.Anything,
		mock.Anything).Return(expectedErr).Twice()

	_, err = api.DeleteWorkspaceConfigPolicies(ctx,
		&apiv1.DeleteWorkspaceConfigPoliciesRequest{
			WorkspaceId:  workspaceID,
			WorkloadType: model.NTSCType,
		})
	require.Equal(t, expectedErr, err)

	_, err = api.PutWorkspaceConfigPolicies(ctx,
		&apiv1.PutWorkspaceConfigPoliciesRequest{
			WorkspaceId:    workspaceID,
			WorkloadType:   model.NTSCType,
			ConfigPolicies: validConstraintsPolicyYAML,
		})
	require.Equal(t, expectedErr, err)

	// (Workspace-level) Nil error returns whatever the request returned.
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Twice()
	configPolicyAuthZ.On("CanModifyWorkspaceConfigPolicies", mock.Anything, mock.Anything,
		mock.Anything).Return(nil).Twice()

	_, err = api.DeleteWorkspaceConfigPolicies(ctx,
		&apiv1.DeleteWorkspaceConfigPoliciesRequest{
			WorkspaceId:  workspaceID,
			WorkloadType: model.NTSCType,
		})
	require.NoError(t, err)

	_, err = api.PutWorkspaceConfigPolicies(ctx,
		&apiv1.PutWorkspaceConfigPoliciesRequest{
			WorkspaceId:    workspaceID,
			WorkloadType:   model.NTSCType,
			ConfigPolicies: validConstraintsPolicyYAML,
		})
	require.NoError(t, err)

	// (Global) Deny with permission access error.
	expectedErr = fmt.Errorf("canModifyGlobalConfigPoliciesError")
	configPolicyAuthZ.On("CanModifyGlobalConfigPolicies", mock.Anything, mock.Anything).
		Return(expectedErr, nil).Twice()

	_, err = api.DeleteGlobalConfigPolicies(ctx,
		&apiv1.DeleteGlobalConfigPoliciesRequest{WorkloadType: model.NTSCType})
	require.Equal(t, expectedErr, err)

	_, err = api.PutGlobalConfigPolicies(ctx,
		&apiv1.PutGlobalConfigPoliciesRequest{
			WorkloadType:   model.NTSCType,
			ConfigPolicies: validNTSCConfigPolicyYAML,
		})
	require.Equal(t, expectedErr, err)

	// (Global) Nil error returns whatever the request returned.
	configPolicyAuthZ.On("CanModifyGlobalConfigPolicies", mock.Anything, mock.Anything).
		Return(nil, nil).Twice()

	_, err = api.DeleteGlobalConfigPolicies(ctx,
		&apiv1.DeleteGlobalConfigPoliciesRequest{WorkloadType: model.NTSCType})
	require.NoError(t, err)

	_, err = api.PutGlobalConfigPolicies(ctx,
		&apiv1.PutGlobalConfigPoliciesRequest{
			WorkloadType:   model.NTSCType,
			ConfigPolicies: validNTSCConfigPolicyYAML,
		})
	require.NoError(t, err)
}

var cpAuthZ *mocks.ConfigPolicyAuthZ

func setupConfigPolicyAuthZ() *mocks.ConfigPolicyAuthZ {
	if cpAuthZ == nil {
		cpAuthZ = &mocks.ConfigPolicyAuthZ{}
		configpolicy.AuthZProvider.Register("mock", cpAuthZ)
	}
	return cpAuthZ
}

func TestValidatePoliciesAndWorkloadTypeYAML(t *testing.T) {
	tests := []struct {
		name           string
		workloadType   string
		configPolicies string
		err            error
	}{
		{
			"YAML invalid workload type valid config policies", "random", validExperimentConfigPolicyYAML,
			fmt.Errorf(invalidWorkloadTypeErr),
		},
		{
			"YAML no workload type valid config policies", "", validExperimentConfigPolicyYAML,
			fmt.Errorf(noWorkloadErr),
		},
		{
			"YAML valid workload type no config policies", model.ExperimentType, "",
			fmt.Errorf(noPoliciesErr),
		},

		// Valid experiment invariant config policies (YAML).
		{
			"YAML simple experiment config with description", model.ExperimentType,
			`
invariant_config:
  description: "test\nspecial\tchar"
`, nil,
		},
		{
			"YAML simple experiment config with resources", model.ExperimentType,
			`
invariant_config:
  resources:
    slots: 1
`, nil,
		},
		{"YAML partial experiment config", model.ExperimentType, validExperimentConfigPolicyYAML, nil},

		// Valid NTSC invariant config policies (YAML).
		{
			"YAML simple NTSC config with description", model.NTSCType,
			`
invariant_config:
  description: "test\nspecial\tchar"
`, nil,
		},
		{
			"YAML simple NTSC config with resources", model.NTSCType,
			`
invariant_config:
  resources:
    slots: 1
`, nil,
		},
		{"YAML partial NTSC config", model.NTSCType, validNTSCConfigPolicyYAML, nil},

		// Invalid experiment invariant config policies (YAML).
		{
			"YAML experiment config with null key", model.ExperimentType,
			`
invariant_config:
  : "test\nspecial\tchar"
`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"YAML experiment config invalid type for key", model.ExperimentType,
			`
invariant_config:
  resources:
    slots: "this should be a number"
`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"YAML experiment config nonexistent key", model.ExperimentType,
			`
invariant_config:
  nonexistent_description: "test\nspecial\tchar"
`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"YAML experiment config extra nonexistent key", model.ExperimentType,
			`
invariant_config:
  resources:
    slots: 1
  extra_key: 2
`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"YAML experiment just config", model.ExperimentType,
			`
invariant_config:
`, fmt.Errorf(configpolicy.EmptyInvariantConfigErr),
		},
		{
			"YAML experiment bad config spec", model.ExperimentType,
			`
bad_config_spec:
  resources:
    slots: 1
`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},

		// Invalid NTSC invariant config policies (YAML).
		{
			"YAML NTSC config with null key", model.NTSCType,
			`
invariant_config:
  : "test\nspecial\tchar"
`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"YAML NTSC invalid NTSC type for key", model.NTSCType,
			`
invariant_config:
  resources:
    slots: "this should be a number"
`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"YAML NTSC nonexistent key", model.NTSCType,
			`
invariant_config:
  nonexistent_description: "test\nspecial\tchar"
`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"YAML NTSC extra nonexistent key", model.NTSCType,
			`
invariant_config:
  resources:
    slots: 1
  extra_key: 2
`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"YAML NTSC just config", model.NTSCType,
			`
invariant_config:
`, fmt.Errorf(configpolicy.EmptyInvariantConfigErr),
		},
		{
			"YAML NTSC bad config spec", model.NTSCType,
			`
bad_config_spec:
  resources:
    slots: 1
`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		// Valid constraint policies (YAML).
		{"YAML experiment valid constraints policy", model.ExperimentType, validConstraintsPolicyYAML, nil},
		{"YAML NTSC valid constraints policy", model.NTSCType, validConstraintsPolicyYAML, nil},
		{
			"YAML experiment simple valid constraints policy priority limit", model.ExperimentType,
			`
constraints:
  priority_limit: 10
`, nil,
		},
		{
			"YAML NTSC simple valid constraints policy priority limit", model.NTSCType,
			`
constraints:
  priority_limit: 10
`, nil,
		},
		{
			"YAML experiment simple valid constraints policy resources", model.ExperimentType,
			`
constraints:
  resources:
    max_slots: 4
`, nil,
		},
		{
			"YAML NTSC simple valid constraints policy resources", model.NTSCType,
			`
constraints:
  resources:
    max_slots: 4
`, nil,
		},

		// Invalid experiment constraint policies (YAML).
		{
			"experiment constraint with null key", model.ExperimentType,
			`
constraints:
 : 10
`,
			fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"YAML experiment constraints invalid type for key", model.ExperimentType,
			`
constraints:
  resources:
    max_slots: "this should be a number"
`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"YAML experiment constraints nonexistent key", model.ExperimentType,
			`
constraints:
  nonexistent_priority_limit: 10
`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"YAML experiment constraints extra nonexistent key", model.ExperimentType,
			`
constraints:
  resources:
    max_slots: 1
  extra_key: 2
`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"YAML experiment just constraints", model.ExperimentType,
			`
constraints:
`, fmt.Errorf(configpolicy.EmptyInvariantConfigErr),
		},

		// Invalid NTSC constraint policies (YAML).
		{
			"YAML NTSC constraint with null key", model.NTSCType,
			`
constraints:
: 10
`,
			fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"YAML NTSC constraints invalid type for key", model.NTSCType,
			`
constraints:
  resources:
    max_slots: "this should be a number"
`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"YAML NTSC constraints nonexistent key", model.NTSCType,
			`
constraints:
  nonexistent_priority_limit: 10
`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"YAML NTSC constraints extra nonexistent key", model.NTSCType,
			`
constraints:
  resources:
    max_slots: 1
  extra_key: 2
`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"YAML NTSC just constraints", model.NTSCType,
			`
constraints:
`, fmt.Errorf(configpolicy.EmptyInvariantConfigErr),
		},

		// Additional experiment combinatory tests (YAML).
		{
			"YAML experiment valid config valid constraints", model.ExperimentType,
			validExperimentConfigPolicyYAML + validConstraintsPolicyYAML, nil,
		},
		{
			"YAML experiment valid constraints invalid constraints", model.ExperimentType,
			validExperimentConfigPolicyYAML + `
constraints:
  resources:
    max_slots: "this should be a number"
`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"YAML experiment invalid config valid constraints", model.ExperimentType,
			`
invariant_config:
  resources:
    slots: "this should be a number"
` + validConstraintsPolicyYAML, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},

		// Additional NTSC combinatory tests (YAML).
		{
			"YAML NTSC valid config valid constraints", model.NTSCType,
			validNTSCConfigPolicyYAML + validConstraintsPolicyYAML, nil,
		},
		{
			"YAML NTSC valid constraints invalid constraints", model.NTSCType,
			validNTSCConfigPolicyYAML + `
constraints:
  resources:
    max_slots: "this should be a number"
`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"NTSC invalid config valid constraints", model.NTSCType,
			`
invariant_config:
  resources:
    slots: "this should be a number"
` + validConstraintsPolicyYAML, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},

		// Experiment-NTSC mismatch test error (YAML).
		{
			"YAML valid experiment config with NTSC workload type", model.NTSCType,
			validExperimentConfigPolicyYAML, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"YAML valid NTSC config with experiment workload type", model.ExperimentType,
			validNTSCConfigPolicyYAML, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePoliciesAndWorkloadType(test.workloadType, test.configPolicies)
			if test.err != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, test.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePoliciesAndWorkloadTypeJSON(t *testing.T) {
	tests := []struct {
		name           string
		workloadType   string
		configPolicies string
		err            error
	}{
		// Valid experiment invariant config policies.
		{
			"JSON simple experiment config with description", model.ExperimentType,
			`{ "invariant_config": {
		 "description": "test\nspecial\tchar"
		}
	}`, nil,
		},
		{
			"JSON simple experiment config with resources", model.ExperimentType,
			`{ "invariant_config": {
			 "resources": {
				 "slots": 1 }
			}
	}`, nil,
		},
		{"JSON partial experiment config", model.ExperimentType, "{" +
			validExperimentConfigPolicyJSON + "}", nil},

		// Valid NTSC invariant config policies (JSON).
		{
			"JSON simple NTSC config with description", model.NTSCType,
			`{ "invariant_config": {
		"description": "test\nspecial\tchar"
		}
	}`, nil,
		},
		{
			"JSON simple NTSC config with resources", model.NTSCType,
			`{ "invariant_config": {
			"resources": {
				"slots": 1
			}
		}
	}`, nil,
		},
		{"JSON partial NTSC config", model.NTSCType, "{" + validNTSCConfigPolicyJSON + "}", nil},

		// Invalid experiment invariant config policies (JSON).
		{
			"JSON experiment config with null key", model.ExperimentType,
			`{ "invariant_config":
			  : "test\nspecial\tchar"
	}`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"JSON experiment config invalid type for key", model.ExperimentType,
			`{ "invariant_config:
		  "resources": {
			slots: "this should be a number"
		}
	}`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"JSON experiment config nonexistent key", model.ExperimentType,
			`{ "invariant_config":
		"nonexistent_description": "test\nspecial\tchar" 
}`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"JSON experiment config extra nonexistent key", model.ExperimentType,
			`{ "invariant_config": {
			"resources": {
				"slots": 1
			},
			"extra_key": 2
		}
	}`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"JSON experiment just config", model.ExperimentType,
			`{"invariant_config": }`, fmt.Errorf(configpolicy.EmptyInvariantConfigErr),
		},
		{
			"JSON experiment bad config spec", model.ExperimentType,
			`{ "bad_config_spec": {
			"resources": {
				"slots": 1
			}				}
	}`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},

		// Invalid NTSC invariant config policies (JSON).
		{
			"JSON NTSC config with null key", model.NTSCType,
			`{ "invariant_config": {
		   : "test\nspecial\tchar"
	 }`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"JSON NTSC config with empty key", model.NTSCType,
			`{ "invariant_config": {
		   "": "test\nspecial\tchar"
	 }`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"JSON NTSC invalid NTSC type for key", model.NTSCType,
			`{ "invariant_config":
		"resources": {
			"slots": "this should be a number"
		}`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"JSON NTSC nonexistent key", model.NTSCType,
			`{ "invariant_config": {
		   "nonexistent_description": "test\nspecial\tchar"
	 }`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"JSON NTSC extra nonexistent key", model.NTSCType,
			`{ "invariant_config": {
			"resources": {
				 "slots": 1
		  },
		"extra_key": 2
	}`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"JSON NTSC just config", model.NTSCType,
			`{ "invariant_config": }`, fmt.Errorf(configpolicy.EmptyInvariantConfigErr),
		},
		{
			"JSON NTSC bad config spec", model.NTSCType,
			`{ "bad_config_spec": {
			"resources": {
				"slots": 1
			}				
		}
	}`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		// Valid constraint policies (JSON).
		{
			"JSON experiment valid constraints policy", model.ExperimentType,
			"{" + validConstraintsPolicyJSON + "}", nil,
		},
		{"JSON NTSC valid constraints policy", model.NTSCType, "{" + validConstraintsPolicyJSON +
			"}", nil},
		{
			"JSON experiment simple valid constraints policy priority limit", model.ExperimentType,
			`{ "constraints": {
			"priority_limit": 10
		}
	}`, nil,
		},
		{
			"JSON NTSC simple valid constraints policy priority limit", model.NTSCType,
			`{ "constraints": {
			"priority_limit": 10
		}
	}`, nil,
		},
		{
			"JSON experiment simple valid constraints policy resources", model.ExperimentType,
			`{ "constraints": {
			"resources": {
				"max_slots": 4
			}
		}
	}`, nil,
		},
		{
			"JSON NTSC simple valid constraints policy resources", model.NTSCType,
			`{ "constraints": {
			"resources": {
				"max_slots": 4
			}
		}
	}`, nil,
		},

		// Invalid experiment constraint policies (JSON).
		{
			"JSON experiment constraint with null key", model.ExperimentType,
			`{ "constraints": {
		: 10 
		}
	}`,
			fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"JSON experiment constraints invalid type for key", model.ExperimentType,
			`{ "constraints": {
			"resources": {
				"max_slots": "this should be a number"
			}
	}`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"JSON experiment constraints nonexistent key", model.ExperimentType,
			`{ "constraints": {
		"nonexistent_priority_limit": 10
	}`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"JSON experiment constraints extra nonexistent key", model.ExperimentType,
			`{ "constraints": {
			"resources": {
				"max_slots": 1
			},
			"extra_key": 2 
	}`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"JSON experiment just constraints", model.ExperimentType,
			`{ "constraints": }`, fmt.Errorf(configpolicy.EmptyInvariantConfigErr),
		},

		// Invalid NTSC constraint policies (JSON).
		{
			"JSON NTSC constraint with null key", model.ExperimentType,
			`{ "constraints": {
		: 10 
		}
	}`,
			fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"JSON NTSC constraint with empty key", model.ExperimentType,
			`{ "constraints": {
		"": 10 
		}
	}`,
			fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"JSON NTSC constraints invalid type for key", model.NTSCType,
			`"constraints": {
		"resources": {
			"max_slots": "this should be a number"
		}
	}`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"JSON NTSC constraints nonexistent key", model.NTSCType,
			`{ "constraints":
			  "nonexistent_priority_limit": 10 
	  }`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"JSON NTSC constraints extra nonexistent key", model.NTSCType,
			`{ "constraints": {
		"resources": {
			"max_slots": 1
		},
	  "extra_key": 2
	}`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"JSON NTSC just constraints", model.NTSCType,
			`{ "constraints": }`, fmt.Errorf(configpolicy.EmptyInvariantConfigErr),
		},

		// Additional experiment combinatory tests (JSON).
		{
			"JSON experiment valid config valid constraints", model.ExperimentType,
			"{" + validExperimentConfigPolicyJSON + "," + validConstraintsPolicyJSON + "}", nil,
		},
		{
			"JSON experiment valid constraints invalid constraints", model.ExperimentType,
			"{" + validExperimentConfigPolicyJSON + "," +
				` 
		"constraints: {
			"resources": {
				"max_slots": "this should be a number" 
			}
		}
	}`, fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			"JSON experiment invalid config valid constraints", model.ExperimentType,
			`{ "invariant_config": {
			"resources": {
				"slots": "this should be a number"
			}
	},` + validConstraintsPolicyJSON + "}", fmt.Errorf(invalidExperimentConfigPolicyErr),
		},

		// Additional NTSC combinatory tests (JSON).
		{
			"JSON NTSC valid config valid constraints", model.NTSCType,
			"{" + validNTSCConfigPolicyJSON + "," + validConstraintsPolicyJSON + "}", nil,
		},
		{
			"JSON NTSC valid constraints invalid constraints", model.NTSCType,
			"{" + validNTSCConfigPolicyJSON + "," +
				`
		"constraints": {
			"resources": {
				"max_slots": "this should be a number"
			}
		}
	}`, fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"JSON NTSC invalid config valid constraints", model.NTSCType,
			`{ "invariant_config": {
			"resources": {
				"slots": "this should be a number"
			}
	},
` + validConstraintsPolicyJSON + "}", fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},

		// Experiment-NTSC mismatch test error (JSON).
		{
			"JSON valid experiment config with NTSC workload type", model.NTSCType,
			"{" + validExperimentConfigPolicyJSON + "}", fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
		{
			"JSON valid NTSC config with experiment workload type", model.ExperimentType,
			"{" + validNTSCConfigPolicyJSON + "}", fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePoliciesAndWorkloadType(test.workloadType, test.configPolicies)
			if test.err != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, test.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParseConfigPolicies(t *testing.T) {
	validExperimentConfigYAML := `
description: "test\nspecial\tchar"
environment:
  force_pull_image: false
  add_capabilities:
    - "cap1"
    - "cap2"
resources:
  slots: 1
name: my_experiment_config
`
	validConstraintsYAML := `
resources:
  max_slots: 4
priority_limit: 10
`

	validNTSCConfigYAML := `
description: "test\nspecial\tchar"
environment:
  force_pull_image: false
  add_capabilities:
    - "cap1"
    - "cap2"
resources:
  slots: 1
work_dir: my/working/directory
`
	validExperimentConfigJSON := "{" + validExperimentConfigPolicyJSON + "}"
	validConstraintsJSON := "{" + validConstraintsPolicyJSON + "}"

	// We create experiment and NTSC config and constraints maps to generate the expected
	// configuration without replicating the marshaling code used in the parseConfigPolicies
	// function to generate our expected output. So we compare configuration maps instead of the
	// directly outputted strings since identical maps imply identical configuration in differently
	// formatted strings.
	// This further assures correct function logic since our expected values were derived without
	// replicating any function logic.
	var validExperimentConfigMap map[string]interface{}
	err := yaml.Unmarshal([]byte(validExperimentConfigYAML), &validExperimentConfigMap)
	require.NoError(t, err)

	var validConstraintsMap map[string]interface{}
	err = yaml.Unmarshal([]byte(validConstraintsYAML), &validConstraintsMap)
	require.NoError(t, err)

	var validNTSCConfigMap map[string]interface{}
	err = yaml.Unmarshal([]byte(validNTSCConfigYAML), &validNTSCConfigMap)
	require.NoError(t, err)

	// We cast the integers to float64 throughout the tcps maps because yaml.Unmarshal decodes all
	// objects of numbers type into float64 by default.
	tests := []struct {
		name            string
		configPolicies  string
		tcps            map[string]interface{}
		invariantConfig map[string]interface{}
		constraints     map[string]interface{}
	}{
		{
			"valid YAML exp config and constraints", validExperimentConfigPolicyYAML +
				validConstraintsPolicyYAML,
			map[string]interface{}{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{"slots": float64(1)},
					"name":      "my_experiment_config",
				},
				"constraints": map[string]interface{}{
					"resources":      map[string]interface{}{"max_slots": float64(4)},
					"priority_limit": float64(10),
				},
			},
			validExperimentConfigMap, validConstraintsMap,
		},

		{
			"valid JSON experiment config and constraints", "{" + validExperimentConfigPolicyJSON +
				"," + validConstraintsPolicyJSON + "}",
			map[string]interface{}{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},

					"resources": map[string]interface{}{"slots": float64(1)},
					"name":      "my_experiment_config",
				},
				"constraints": map[string]interface{}{
					"resources":      map[string]interface{}{"max_slots": float64(4)},
					"priority_limit": float64(10),
				},
			},
			validExperimentConfigMap, validConstraintsMap,
		},
		{
			"valid YAML NTSC config and constraints", validNTSCConfigPolicyYAML +
				validConstraintsPolicyYAML,
			map[string]interface{}{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{"slots": float64(1)},
					"work_dir":  "my/working/directory",
				},
				"constraints": map[string]interface{}{
					"resources":      map[string]interface{}{"max_slots": float64(4)},
					"priority_limit": float64(10),
				},
			},
			validNTSCConfigMap, validConstraintsMap,
		},
		{
			"valid JSON NTSC config and constraints", "{" + validNTSCConfigPolicyJSON +
				"," + validConstraintsPolicyJSON + "}",
			map[string]interface{}{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},

					"resources": map[string]interface{}{"slots": float64(1)},
					"work_dir":  "my/working/directory",
				},
				"constraints": map[string]interface{}{
					"resources":      map[string]interface{}{"max_slots": float64(4)},
					"priority_limit": float64(10),
				},
			},
			validNTSCConfigMap, validConstraintsMap,
		},
		{
			"just constraints YAML", validConstraintsPolicyYAML,
			map[string]interface{}{
				"constraints": map[string]interface{}{
					"resources":      map[string]interface{}{"max_slots": float64(4)},
					"priority_limit": float64(10),
				},
			},
			nil, validConstraintsMap,
		},
		{
			"just constraints JSON", validConstraintsJSON, map[string]interface{}{
				"constraints": map[string]interface{}{
					"resources":      map[string]interface{}{"max_slots": float64(4)},
					"priority_limit": float64(10),
				},
			}, nil, validConstraintsMap,
		},
		{
			"just experiment config YAML", validExperimentConfigPolicyYAML,
			map[string]interface{}{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},

					"resources": map[string]interface{}{"slots": float64(1)},
					"name":      "my_experiment_config",
				},
			},
			validExperimentConfigMap, nil,
		},
		{
			"just experiment config JSON", validExperimentConfigJSON,
			map[string]interface{}{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},

					"resources": map[string]interface{}{"slots": float64(1)},
					"name":      "my_experiment_config",
				},
			},
			validExperimentConfigMap, nil,
		},
		{
			"just NTSC config YAML", validNTSCConfigPolicyYAML,
			map[string]interface{}{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},

					"resources": map[string]interface{}{"slots": float64(1)},
					"work_dir":  "my/working/directory",
				},
			},
			validNTSCConfigMap, nil,
		},
		{
			"just NTSC config JSON", "{" + validNTSCConfigPolicyJSON + "}",
			map[string]interface{}{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},

					"resources": map[string]interface{}{"slots": float64(1)},
					"work_dir":  "my/working/directory",
				},
			},
			validNTSCConfigMap, nil,
		},
		{
			"random valid YAML with neither config nor constraint",
			`
a_key: "a_value"
another_key:
  sub_key: 1
`,
			map[string]interface{}{
				"a_key":       "a_value",
				"another_key": map[string]interface{}{"sub_key": float64(1)},
			},
			nil, nil,
		},
		{
			"random valid JSON with neither config nor constraint",

			`{
			"a_key": "a_value",
			"another_key": {
				"sub_key": 1
				}
			}`,
			map[string]interface{}{
				"a_key":       "a_value",
				"another_key": map[string]interface{}{"sub_key": float64(1)},
			},
			nil, nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tcps, invariantConfig, constraints, err := parseConfigPolicies(test.configPolicies)
			require.NoError(t, err)

			// Verify invariant config output is correct
			if test.invariantConfig != nil {
				require.NotNil(t, invariantConfig)

				var invariantConfigMap map[string]interface{}
				err = yaml.Unmarshal([]byte(*invariantConfig), &invariantConfigMap)
				require.NoError(t, err)
				require.Equal(t, test.invariantConfig, invariantConfigMap)
			} else {
				require.Nil(t, invariantConfig)
			}

			// Verify constraints output is correct.
			if test.constraints != nil {
				require.NotNil(t, constraints)

				var constraintsMap map[string]interface{}
				err = yaml.Unmarshal([]byte(*constraints), &constraintsMap)
				require.NoError(t, err)
				require.Equal(t, test.constraints, constraintsMap)
			} else {
				require.Nil(t, constraints)
			}

			// Verify map output is correct.
			require.Equal(t, test.tcps, tcps)
		})
	}

	// Test empty string fails.
	tcps, invariantConfig, constraints, err := parseConfigPolicies("")
	require.Nil(t, tcps)
	require.Nil(t, invariantConfig)
	require.Nil(t, constraints)
	require.Error(t, err)
	require.ErrorContains(t, err, "nothing to parse, empty config and constraints input")

	// Test non-JSON and non-YAML formatted strings fail.
	tcps, invariantConfig, constraints, err = parseConfigPolicies("{\"bad_json\",: 1}")
	require.Nil(t, tcps)
	require.Nil(t, invariantConfig)
	require.Nil(t, constraints)
	require.Error(t, err)
	require.ErrorContains(t, err, "error parsing config policies")

	tcps, invariantConfig, constraints, err = parseConfigPolicies("bad_yaml:1")
	require.Nil(t, tcps)
	require.Nil(t, invariantConfig)
	require.Nil(t, constraints)
	require.Error(t, err)
	require.ErrorContains(t, err, "error unmarshaling config policies")

	tcps, invariantConfig, constraints, err = parseConfigPolicies("random string")
	require.Nil(t, tcps)
	require.Nil(t, invariantConfig)
	require.Nil(t, constraints)
	require.Error(t, err)
	require.ErrorContains(t, err, "error unmarshaling config policies")
}

func TestPutWorkspaceConfigPolicies(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	testutils.MustLoadLicenseAndKeyFromFilesystem("../../")

	workspaceIDs := []int32{}
	defer func() {
		err := db.CleanupMockWorkspace(workspaceIDs)
		if err != nil {
			log.Errorf("error when cleaning up mock workspaces")
		}
	}()

	tests := []struct {
		name                  string
		req                   *apiv1.PutWorkspaceConfigPoliciesRequest
		configPolicies        map[string]any
		updatedPoliciesReq    *apiv1.PutWorkspaceConfigPoliciesRequest
		updatedConfigPolicies map[string]any
		err                   error
	}{
		{
			name: "invalid workload type",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType:   "bad type",
				ConfigPolicies: validExperimentConfigPolicyYAML,
			},
			configPolicies:        nil,
			updatedPoliciesReq:    nil,
			updatedConfigPolicies: nil,
			err:                   fmt.Errorf("invalid workload type"),
		},
		{
			name: "empty workload type",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				ConfigPolicies: validExperimentConfigPolicyYAML,
			},
			configPolicies:        nil,
			updatedPoliciesReq:    nil,
			updatedConfigPolicies: nil,
			err:                   fmt.Errorf("no workload type"),
		},
		{
			name: "valid experiment invariant config add update YAML",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType:   model.ExperimentType,
				ConfigPolicies: validExperimentConfigPolicyYAML,
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name": "my_experiment_config",
				},
			},
			updatedPoliciesReq: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType:   model.ExperimentType,
				ConfigPolicies: validExperimentConfigPolicyYAML + entrypointYAML,
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name":       "my_experiment_config",
					"entrypoint": "start from here",
				},
			},
			err: nil,
		},
		{
			name: "valid experiment invariant config add update JSON",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType:   model.ExperimentType,
				ConfigPolicies: "{" + validExperimentConfigPolicyJSON + "}",
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name": "my_experiment_config",
				},
			},
			updatedPoliciesReq: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType:   model.ExperimentType,
				ConfigPolicies: "{" + updatedExperimentConfigPolicyJSON + "}",
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name":       "my_experiment_config",
					"entrypoint": "start from here",
				},
			},
			err: nil,
		},
		{
			name: "simple valid experiment config YAML",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
invariant_config:
  description: "test\nspecial\tchar"
  name: my_experiment_config
`,
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"name":        "my_experiment_config",
				},
			},
			updatedPoliciesReq: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
invariant_config:
  description: "new description!"
  name: my_experiment_config
`,
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "new description!",
					"name":        "my_experiment_config",
				},
			},
			err: nil,
		},
		{
			name: "simple valid experiment config JSON",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{ 
				"invariant_config": {
					"description": "test\nspecial\tchar",
					"name": "my_experiment_config"
					}
				}`,
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"name":        "my_experiment_config",
				},
			},
			updatedPoliciesReq: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{ 
				"invariant_config": {
					"description": "new description!",
					"name": "my_experiment_config"
					}
				}`,
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "new description!",
					"name":        "my_experiment_config",
				},
			},
			err: nil,
		},
		{
			name: "simple valid experiment constraints delete and modify update YAML",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
constraints:
  resources:
    max_slots: 10
  priority_limit: 20
`,
			},
			configPolicies: map[string]any{
				"constraints": map[string]interface{}{
					"resources":      map[string]interface{}{"max_slots": float64(10)},
					"priority_limit": float64(20),
				},
			},
			updatedPoliciesReq: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
constraints:
  priority_limit: 30
`,
			},
			updatedConfigPolicies: map[string]any{
				"constraints": map[string]interface{}{
					"priority_limit": float64(30),
				},
			},
			err: nil,
		},
		{
			name: "simple valid experiment constraints delete and modify update JSON",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{ 
					"constraints": {
						"resources": {
							"max_slots": 10
						},
						"priority_limit": 20
					}
			}`,
			},
			configPolicies: map[string]any{
				"constraints": map[string]interface{}{
					"resources":      map[string]interface{}{"max_slots": float64(10)},
					"priority_limit": float64(20),
				},
			},
			updatedPoliciesReq: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{ 
					"constraints": {
						"priority_limit": 30
					}
				}`,
			},
			updatedConfigPolicies: map[string]any{
				"constraints": map[string]interface{}{
					"priority_limit": float64(30),
				},
			},
			err: nil,
		},
		{
			name: "experiment config and constraints YAML",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
invariant_config:
  description: "test\nspecial\tchar"
  environment:
    force_pull_image: false
    add_capabilities:
      - cap1
      - cap2
  resources:
    slots: 1
  name: my_experiment_config
constraints:
  resources:
    max_slots: 4
  priority_limit: 10
`,
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name": "my_experiment_config",
				},
				"constraints": map[string]interface{}{
					"resources": map[string]interface{}{
						"max_slots": float64(4),
					},
					"priority_limit": float64(10),
				},
			},
			updatedPoliciesReq: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
invariant_config:
  description: "test\nspecial\tchar"
  environment:
    add_capabilities:
      - cap1
  resources:
    slots: 5
  name: my_experiment_config
  entrypoint: "start from here"
constraints:
  resources:
    max_slots: 8
  priority_limit: 10
`,
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"add_capabilities": []interface{}{"cap1"},
					},
					"resources": map[string]interface{}{
						"slots": float64(5),
					},
					"name":       "my_experiment_config",
					"entrypoint": "start from here",
				},
				"constraints": map[string]interface{}{
					"resources": map[string]interface{}{
						"max_slots": float64(8),
					},
					"priority_limit": float64(10),
				},
			},
			err: nil,
		},
		{
			name: "experiment config and constraints JSON",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{
					"invariant_config": {
						"description": "test\nspecial\tchar",
						"environment": {
							"force_pull_image": false,
							"add_capabilities": ["cap1", "cap2"]
						},
						"resources": {
							"slots": 1
						},
						"name": "my_experiment_config"
					},
					"constraints": {
						"resources": {
							"max_slots": 4
						},
						"priority_limit": 10
					}
				}`,
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name": "my_experiment_config",
				},
				"constraints": map[string]interface{}{
					"resources": map[string]interface{}{
						"max_slots": float64(4),
					},
					"priority_limit": float64(10),
				},
			},
			updatedPoliciesReq: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{
					"invariant_config": {
						"description": "test\nspecial\tchar",
						"environment": {
							"add_capabilities": ["cap1"]
						},
						"resources": {
							"slots": 5
						},
						"name": "my_experiment_config",
						"entrypoint": "start from here",
					},
					"constraints": {
						"resources": {
							"max_slots": 8
						},
						"priority_limit": 10
					}
				}`,
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"add_capabilities": []interface{}{"cap1"},
					},
					"resources": map[string]interface{}{
						"slots": float64(5),
					},
					"name":       "my_experiment_config",
					"entrypoint": "start from here",
				},
				"constraints": map[string]interface{}{
					"resources": map[string]interface{}{
						"max_slots": float64(8),
					},
					"priority_limit": float64(10),
				},
			},
			err: nil,
		},
		{
			name: "invalid constraints JSON",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{
					"constraints": {
						"resources": {
							"max_slots": a_string_not_int
						}					
					}
				}`,
			},
			configPolicies:        nil,
			updatedPoliciesReq:    nil,
			updatedConfigPolicies: nil,
			err:                   fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			name: "invalid NTSC config YAML",
			req: &apiv1.PutWorkspaceConfigPoliciesRequest{
				WorkloadType: model.NTSCType,
				ConfigPolicies: `
invariant_config:
  : "null key"
`,
			},
			configPolicies:        nil,
			updatedPoliciesReq:    nil,
			updatedConfigPolicies: nil,
			err:                   fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wkspResp, err := api.PostWorkspace(ctx,
				&apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
			workspaceID := wkspResp.Workspace.Id
			workspaceIDs = append(workspaceIDs, workspaceID)
			require.NoError(t, err)

			test.req.WorkspaceId = workspaceID
			resp, err := api.PutWorkspaceConfigPolicies(ctx, test.req)
			if test.err != nil {
				require.ErrorContains(t, err, test.err.Error())
				require.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.configPolicies, resp.ConfigPolicies.AsMap())

			// Verify that we can retrieve the input config policies.
			getResp, err := api.GetWorkspaceConfigPolicies(ctx,
				&apiv1.GetWorkspaceConfigPoliciesRequest{
					WorkspaceId:  workspaceID,
					WorkloadType: test.req.WorkloadType,
				})
			require.NoError(t, err)

			configPolicies := getResp.ConfigPolicies.AsMap()
			require.Equal(t, test.configPolicies, configPolicies)

			test.updatedPoliciesReq.WorkspaceId = workspaceID
			resp, err = api.PutWorkspaceConfigPolicies(ctx, test.updatedPoliciesReq)
			require.NoError(t, err)
			require.Equal(t, test.updatedConfigPolicies, resp.ConfigPolicies.AsMap())

			// Verify that config policies were updated correctly.
			getResp, err = api.GetWorkspaceConfigPolicies(ctx,
				&apiv1.GetWorkspaceConfigPoliciesRequest{
					WorkspaceId:  workspaceID,
					WorkloadType: test.req.WorkloadType,
				})
			require.NoError(t, err)

			updatedConfigPolicies := getResp.ConfigPolicies.AsMap()
			require.Equal(t, test.updatedConfigPolicies, updatedConfigPolicies)
		})
	}

	// Test invalid workspace ID
	resp, err := api.PutWorkspaceConfigPolicies(ctx,
		&apiv1.PutWorkspaceConfigPoliciesRequest{
			WorkspaceId:    int32(-1),
			WorkloadType:   model.NTSCType,
			ConfigPolicies: validExperimentConfigPolicyYAML,
		})
	require.Nil(t, resp)
	require.ErrorContains(t, err, "NotFound")
}

func TestPutGlobalConfigPolicies(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	testutils.MustLoadLicenseAndKeyFromFilesystem("../../")

	updatedExperimentConfigPolicyJSON := `
	"invariant_config": {
        "description": "test\nspecial\tchar",
        "environment": {
            "force_pull_image": false,
            "add_capabilities": ["cap1", "cap2"]
        },
        "resources": {
            "slots": 1
        },
		"name": "my_experiment_config",
		"entrypoint": "start from here"
    }
`

	tests := []struct {
		name                  string
		req                   *apiv1.PutGlobalConfigPoliciesRequest
		configPolicies        map[string]any
		updatedPoliciesReq    *apiv1.PutGlobalConfigPoliciesRequest
		updatedConfigPolicies map[string]any
		err                   error
	}{
		{
			name: "valid experiment invariant config add update YAML",
			req: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType:   model.ExperimentType,
				ConfigPolicies: validExperimentConfigPolicyYAML,
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name": "my_experiment_config",
				},
			},
			updatedPoliciesReq: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType:   model.ExperimentType,
				ConfigPolicies: validExperimentConfigPolicyYAML + entrypointYAML,
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name":       "my_experiment_config",
					"entrypoint": "start from here",
				},
			},
			err: nil,
		},
		{
			name: "valid experiment invariant config add update JSON",
			req: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType:   model.ExperimentType,
				ConfigPolicies: "{" + validExperimentConfigPolicyJSON + "}",
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name": "my_experiment_config",
				},
			},
			updatedPoliciesReq: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType:   model.ExperimentType,
				ConfigPolicies: "{" + updatedExperimentConfigPolicyJSON + "}",
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name":       "my_experiment_config",
					"entrypoint": "start from here",
				},
			},
			err: nil,
		},
		{
			name: "simple valid experiment config YAML",
			req: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
invariant_config:
  description: "test\nspecial\tchar"
  name: my_experiment_config
`,
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"name":        "my_experiment_config",
				},
			},
			updatedPoliciesReq: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
invariant_config:
  description: "new description!"
  name: my_experiment_config
`,
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "new description!",
					"name":        "my_experiment_config",
				},
			},
			err: nil,
		},
		{
			name: "simple valid experiment config JSON",
			req: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{ 
				"invariant_config": {
					"description": "test\nspecial\tchar",
					"name": "my_experiment_config"
					}
				}`,
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"name":        "my_experiment_config",
				},
			},
			updatedPoliciesReq: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{ 
				"invariant_config": {
					"description": "new description!",
					"name": "my_experiment_config"
					}
				}`,
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "new description!",
					"name":        "my_experiment_config",
				},
			},
			err: nil,
		},
		{
			name: "simple valid experiment constraints delete and modify update YAML",
			req: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
constraints:
  resources:
    max_slots: 10
  priority_limit: 20
`,
			},
			configPolicies: map[string]any{
				"constraints": map[string]interface{}{
					"resources":      map[string]interface{}{"max_slots": float64(10)},
					"priority_limit": float64(20),
				},
			},
			updatedPoliciesReq: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
constraints:
  priority_limit: 30
`,
			},
			updatedConfigPolicies: map[string]any{
				"constraints": map[string]interface{}{
					"priority_limit": float64(30),
				},
			},
			err: nil,
		},
		{
			name: "simple valid experiment constraints delete and modify update JSON",
			req: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{ 
					"constraints": {
						"resources": {
							"max_slots": 10
						},
						"priority_limit": 20
					}
			}`,
			},
			configPolicies: map[string]any{
				"constraints": map[string]interface{}{
					"resources":      map[string]interface{}{"max_slots": float64(10)},
					"priority_limit": float64(20),
				},
			},
			updatedPoliciesReq: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{ 
					"constraints": {
						"priority_limit": 30
					}
				}`,
			},
			updatedConfigPolicies: map[string]any{
				"constraints": map[string]interface{}{
					"priority_limit": float64(30),
				},
			},
			err: nil,
		},
		{
			name: "experiment config and constraints YAML",
			req: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
invariant_config:
  description: "test\nspecial\tchar"
  environment:
    force_pull_image: false
    add_capabilities:
      - cap1
      - cap2
  resources:
    slots: 1
  name: my_experiment_config
constraints:
  resources:
    max_slots: 4
  priority_limit: 10
`,
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name": "my_experiment_config",
				},
				"constraints": map[string]interface{}{
					"resources": map[string]interface{}{
						"max_slots": float64(4),
					},
					"priority_limit": float64(10),
				},
			},
			updatedPoliciesReq: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `
invariant_config:
  description: "test\nspecial\tchar"
  environment:
    add_capabilities:
      - cap1
  resources:
    slots: 5
  name: my_experiment_config
  entrypoint: "start from here"
constraints:
  resources:
    max_slots: 8
  priority_limit: 10
`,
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"add_capabilities": []interface{}{"cap1"},
					},
					"resources": map[string]interface{}{
						"slots": float64(5),
					},
					"name":       "my_experiment_config",
					"entrypoint": "start from here",
				},
				"constraints": map[string]interface{}{
					"resources": map[string]interface{}{
						"max_slots": float64(8),
					},
					"priority_limit": float64(10),
				},
			},
			err: nil,
		},
		{
			name: "experiment config and constraints JSON",
			req: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{
					"invariant_config": {
						"description": "test\nspecial\tchar",
						"environment": {
							"force_pull_image": false,
							"add_capabilities": ["cap1", "cap2"]
						},
						"resources": {
							"slots": 1
						},
						"name": "my_experiment_config"
					},
					"constraints": {
						"resources": {
							"max_slots": 4
						},
						"priority_limit": 10
					}
				}`,
			},
			configPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"force_pull_image": false,
						"add_capabilities": []interface{}{"cap1", "cap2"},
					},
					"resources": map[string]interface{}{
						"slots": float64(1),
					},
					"name": "my_experiment_config",
				},
				"constraints": map[string]interface{}{
					"resources": map[string]interface{}{
						"max_slots": float64(4),
					},
					"priority_limit": float64(10),
				},
			},
			updatedPoliciesReq: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{
					"invariant_config": {
						"description": "test\nspecial\tchar",
						"environment": {
							"add_capabilities": ["cap1"]
						},
						"resources": {
							"slots": 5
						},
						"name": "my_experiment_config",
						"entrypoint": "start from here",
					},
					"constraints": {
						"resources": {
							"max_slots": 8
						},
						"priority_limit": 10
					}
				}`,
			},
			updatedConfigPolicies: map[string]any{
				"invariant_config": map[string]interface{}{
					"description": "test\nspecial\tchar",
					"environment": map[string]interface{}{
						"add_capabilities": []interface{}{"cap1"},
					},
					"resources": map[string]interface{}{
						"slots": float64(5),
					},
					"name":       "my_experiment_config",
					"entrypoint": "start from here",
				},
				"constraints": map[string]interface{}{
					"resources": map[string]interface{}{
						"max_slots": float64(8),
					},
					"priority_limit": float64(10),
				},
			},
			err: nil,
		},
		{
			name: "invalid constraints JSON",
			req: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.ExperimentType,
				ConfigPolicies: `{
					"constraints": {
						"resources": {
							"max_slots": a_string_not_int
						}					
					}
				}`,
			},
			configPolicies:        nil,
			updatedPoliciesReq:    nil,
			updatedConfigPolicies: nil,
			err:                   fmt.Errorf(invalidExperimentConfigPolicyErr),
		},
		{
			name: "invalid NTSC config YAML",
			req: &apiv1.PutGlobalConfigPoliciesRequest{
				WorkloadType: model.NTSCType,
				ConfigPolicies: `
invariant_config:
  : "null key"
`,
			},
			configPolicies:        nil,
			updatedPoliciesReq:    nil,
			updatedConfigPolicies: nil,
			err:                   fmt.Errorf(invalidNTSCtConfigPolicyErr),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := api.DeleteGlobalConfigPolicies(ctx,
				&apiv1.DeleteGlobalConfigPoliciesRequest{WorkloadType: test.req.WorkloadType})
			require.NoError(t, err)

			resp, err := api.PutGlobalConfigPolicies(ctx, test.req)
			if test.err != nil {
				require.ErrorContains(t, err, test.err.Error())
				require.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.configPolicies, resp.ConfigPolicies.AsMap())

			// Verify that we can retrieve the input config policies.
			getResp, err := api.GetGlobalConfigPolicies(ctx,
				&apiv1.GetGlobalConfigPoliciesRequest{
					WorkloadType: test.req.WorkloadType,
				})
			require.NoError(t, err)

			configPolicies := getResp.ConfigPolicies.AsMap()
			require.Equal(t, test.configPolicies, configPolicies)

			resp, err = api.PutGlobalConfigPolicies(ctx, test.updatedPoliciesReq)
			require.NoError(t, err)
			require.Equal(t, test.updatedConfigPolicies, resp.ConfigPolicies.AsMap())

			// Verify that config policies were updated correctly.
			getResp, err = api.GetGlobalConfigPolicies(ctx,
				&apiv1.GetGlobalConfigPoliciesRequest{
					WorkloadType: test.req.WorkloadType,
				})
			require.NoError(t, err)

			updatedConfigPolicies := getResp.ConfigPolicies.AsMap()
			require.Equal(t, test.updatedConfigPolicies, updatedConfigPolicies)
		})
	}

	// Test invalid workload type.
	resp, err := api.PutGlobalConfigPolicies(ctx, &apiv1.PutGlobalConfigPoliciesRequest{
		WorkloadType:   "bad type",
		ConfigPolicies: validExperimentConfigPolicyYAML,
	})
	require.ErrorContains(t, err, "invalid workload type")
	require.Nil(t, resp)

	// Test empty workload type.
	resp, err = api.PutGlobalConfigPolicies(ctx, &apiv1.PutGlobalConfigPoliciesRequest{
		ConfigPolicies: validExperimentConfigPolicyYAML,
	})
	require.ErrorContains(t, err, "no workload type")
	require.Nil(t, resp)
}

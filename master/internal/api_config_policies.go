package internal

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	"github.com/determined-ai/determined/master/internal/configpolicy"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/license"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func stubData() (*structpb.Struct, error) {
	const yamlString = `
invariant_config:
  description: "test"
constraints:
  resources:
    max_slots: 4
  priority_limit: 10
 `
	// put yaml string into a map
	var yamlMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlString), &yamlMap); err != nil {
		return nil, fmt.Errorf("unable to unmarshal yaml: %w", err)
	}

	return configpolicy.MarshalConfigPolicy(yamlMap), nil
}

/* Test:
- user auth: admins can and non-admins can't
- graceful with non-existent workspace
- graceful with non-existent workload type
- happy path: policies stored (exp & ntsc)
- error path: nothing stored
*/

// Add or update workspace task config policies.
func (a *apiServer) PutWorkspaceConfigPolicies(
	ctx context.Context, req *apiv1.PutWorkspaceConfigPoliciesRequest,
) (*apiv1.PutWorkspaceConfigPoliciesResponse, error) {
	// TODO do we want to wrap errors?

	// Check license; task config policies is EE-only.
	license.RequireLicense("task config policies")

	// Get user.
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	// Confirm workspace exists.
	wksp, err := a.GetWorkspaceByID(ctx, req.WorkspaceId, *curUser, true) // Do we want rejectImmutable=true/false?
	if err != nil {
		return nil, err
	}

	// Check if user is an admin, or admin of the workspace.
	if err = workspace.AuthZProvider.Get().CanModifyWorkspaceConfigPolicies(ctx, *curUser, wksp); err != nil {
		return nil, err
	}

	// Validate user input: valid workload type and valid config policies.
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}

	workloadType := model.WorkloadType(req.WorkloadType)
	var res apiv1.PutWorkspaceConfigPoliciesResponse
	switch workloadType {
	case model.NTSCType:
		// TODO function me
		policies, err := configpolicy.UnmarshalNTSCConfigPolicy(req.ConfigPolicies)
		if err != nil {
			return nil, err
		}

		// TODO update database

		res.ConfigPolicies = configpolicy.MarshalConfigPolicy(policies)

	case model.ExperimentType:
		// TODO function me
		policies, err := configpolicy.UnmarshalExperimentConfigPolicy(req.ConfigPolicies)
		if err != nil {
			return nil, err
		}

		// TODO update database

		res.ConfigPolicies = configpolicy.MarshalConfigPolicy(policies)

	}

	return &res, nil
}

// Add or update global task config policies.
func (a *apiServer) PutGlobalConfigPolicies(
	ctx context.Context, req *apiv1.PutGlobalConfigPoliciesRequest,
) (*apiv1.PutGlobalConfigPoliciesResponse, error) {
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}
	data, err := stubData()
	return &apiv1.PutGlobalConfigPoliciesResponse{ConfigPolicies: data}, err
}

// Get workspace task config policies.
func (a *apiServer) GetWorkspaceConfigPolicies(
	ctx context.Context, req *apiv1.GetWorkspaceConfigPoliciesRequest,
) (*apiv1.GetWorkspaceConfigPoliciesResponse, error) {
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}
	data, err := stubData()
	return &apiv1.GetWorkspaceConfigPoliciesResponse{ConfigPolicies: data}, err
}

// Get global task config policies.
func (a *apiServer) GetGlobalConfigPolicies(
	ctx context.Context, req *apiv1.GetGlobalConfigPoliciesRequest,
) (*apiv1.GetGlobalConfigPoliciesResponse, error) {
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}
	data, err := stubData()
	return &apiv1.GetGlobalConfigPoliciesResponse{ConfigPolicies: data}, err
}

// Delete workspace task config policies.
func (a *apiServer) DeleteWorkspaceConfigPolicies(
	ctx context.Context, req *apiv1.DeleteWorkspaceConfigPoliciesRequest,
) (*apiv1.DeleteWorkspaceConfigPoliciesResponse, error) {
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}
	return &apiv1.DeleteWorkspaceConfigPoliciesResponse{}, nil
}

// Delete global task config policies.
func (a *apiServer) DeleteGlobalConfigPolicies(
	ctx context.Context, req *apiv1.DeleteGlobalConfigPoliciesRequest,
) (*apiv1.DeleteGlobalConfigPoliciesResponse, error) {
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}
	return &apiv1.DeleteGlobalConfigPoliciesResponse{}, nil
}

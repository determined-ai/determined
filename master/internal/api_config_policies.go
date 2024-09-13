package internal

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	"github.com/determined-ai/determined/master/internal/configpolicy"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/master/pkg/model"
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

// Add or update workspace task config policies.
func (*apiServer) PutWorkspaceConfigPolicies(
	ctx context.Context, req *apiv1.PutWorkspaceConfigPoliciesRequest,
) (*apiv1.PutWorkspaceConfigPoliciesResponse, error) {
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}
	data, err := stubData()
	return &apiv1.PutWorkspaceConfigPoliciesResponse{ConfigPolicies: data}, err
}

// Add or update global task config policies.
func (*apiServer) PutGlobalConfigPolicies(
	ctx context.Context, req *apiv1.PutGlobalConfigPoliciesRequest,
) (*apiv1.PutGlobalConfigPoliciesResponse, error) {
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}
	data, err := stubData()
	return &apiv1.PutGlobalConfigPoliciesResponse{ConfigPolicies: data}, err
}

// Get workspace task config policies.
func (*apiServer) GetWorkspaceConfigPolicies(
	ctx context.Context, req *apiv1.GetWorkspaceConfigPoliciesRequest,
) (*apiv1.GetWorkspaceConfigPoliciesResponse, error) {
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}
	
	resp := apiv1.GetWorkspaceConfigPoliciesResponse{}
	if req.WorkloadType == string(model.NTSCType) {
		configPolicies, err := configpolicy.GetNTSCConfigPolicies(ctx, ptrs.Ptr(int(req.WorkspaceId)))
		if err != nil {
			return nil, err
		}
		fmt.Printf("%+v\n", configPolicies)
		type ConfigPolicies struct {
			Constraints model.Constraints
			InvariantConfig model.CommandConfig
		}
		respStruct := ConfigPolicies{Constraints: configPolicies.Constraints, InvariantConfig: configPolicies.InvariantConfig}
		resp.ConfigPolicies = configpolicy.MarshalConfigPolicy(respStruct)
	} else {
		data, err := stubData()
		if err != nil {
			return nil, err
		}
		resp.ConfigPolicies = data
	}
	return &resp, nil
}

// Get global task config policies.
func (*apiServer) GetGlobalConfigPolicies(
	ctx context.Context, req *apiv1.GetGlobalConfigPoliciesRequest,
) (*apiv1.GetGlobalConfigPoliciesResponse, error) {
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}
	data, err := stubData()
	return &apiv1.GetGlobalConfigPoliciesResponse{ConfigPolicies: data}, err
}

// Delete workspace task config policies.
func (*apiServer) DeleteWorkspaceConfigPolicies(
	ctx context.Context, req *apiv1.DeleteWorkspaceConfigPoliciesRequest,
) (*apiv1.DeleteWorkspaceConfigPoliciesResponse, error) {
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}
	return &apiv1.DeleteWorkspaceConfigPoliciesResponse{}, nil
}

// Delete global task config policies.
func (*apiServer) DeleteGlobalConfigPolicies(
	ctx context.Context, req *apiv1.DeleteGlobalConfigPoliciesRequest,
) (*apiv1.DeleteGlobalConfigPoliciesResponse, error) {
	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, fmt.Errorf("invalid workload type: %s", req.WorkloadType)
	}
	return &apiv1.DeleteGlobalConfigPoliciesResponse{}, nil
}

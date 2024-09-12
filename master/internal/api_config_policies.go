package internal

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	"github.com/determined-ai/determined/master/internal/cluster"
	"github.com/determined-ai/determined/master/internal/configpolicy"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
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
	data, err := stubData()
	return &apiv1.GetWorkspaceConfigPoliciesResponse{ConfigPolicies: data}, err
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
func (a *apiServer) DeleteWorkspaceConfigPolicies(
	ctx context.Context, req *apiv1.DeleteWorkspaceConfigPoliciesRequest,
) (*apiv1.DeleteWorkspaceConfigPoliciesResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	err = workspace.AuthZProvider.Get().CanGetWorkspaceID(
		ctx, *curUser, req.WorkspaceId,
	)
	if err != nil {
		return nil, err
	}

	var w model.Workspace
	err = db.Bun().NewSelect().Model(&w).Where("id = ?", req.WorkspaceId).Scan(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	wksp, err := w.ToProto()
	err = workspace.AuthZProvider.Get().CanModifyWorkspaceConfigPolicies(ctx, *curUser, wksp)
	if err != nil {
		return nil, err
	}

	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workload type :%s",
			req.WorkloadType)
	}

	err = configpolicy.DeleteConfigPolicies(ctx, ptrs.Ptr(int(req.WorkspaceId)),
		model.WorkloadType(req.WorkloadType))
	if err != nil {
		return nil, err
	}
	return &apiv1.DeleteWorkspaceConfigPoliciesResponse{}, nil
}

// Delete global task config policies.
func (a *apiServer) DeleteGlobalConfigPolicies(
	ctx context.Context, req *apiv1.DeleteGlobalConfigPoliciesRequest,
) (*apiv1.DeleteGlobalConfigPoliciesResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	permErr, err := cluster.AuthZProvider.Get().CanModifyGlobalConfigPolicies(ctx, curUser)
	if err != nil {
		return nil, err
	} else if permErr != nil {
		return nil, permErr
	}

	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workload type :%s",
			req.WorkloadType)
	}

	err = configpolicy.DeleteConfigPolicies(ctx, nil, model.WorkloadType(req.WorkloadType))
	if err != nil {
		return nil, err
	}
	return &apiv1.DeleteGlobalConfigPoliciesResponse{}, nil
}

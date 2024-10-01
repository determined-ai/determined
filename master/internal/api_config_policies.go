package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ghodss/yaml"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/configpolicy"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/license"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	noWorkloadErr          = "no workload type specified."
	noPoliciesErr          = "no specified config policies."
	invalidWorkloadTypeErr = "invalid workload type"
)

func validatePoliciesAndWorkloadType(workloadType, configPolicies string) error {
	if !configpolicy.ValidWorkloadType(workloadType) {
		errMessage := fmt.Sprintf(invalidWorkloadTypeErr+": %s.", workloadType)
		if len(workloadType) == 0 {
			errMessage = noWorkloadErr
		}
		return status.Errorf(codes.InvalidArgument, errMessage)
	}

	if len(configPolicies) == 0 {
		return status.Errorf(codes.InvalidArgument, noPoliciesErr)
	}

	// Validate the input config based on workload type.
	if workloadType == model.ExperimentType {
		_, err := configpolicy.UnmarshalExperimentConfigPolicy(configPolicies)
		if err != nil {
			return err
		}
	} else {
		_, err := configpolicy.UnmarshalNTSCConfigPolicy(configPolicies)
		if err != nil {
			return err
		}
	}

	// TODO (request validation): Verify that configs do not violate constraints.
	return nil
}

func parseConfigPolicies(configAndConstraints string) (tcps map[string]interface{},
	invariantConfig *string, constraints *string, err error,
) {
	if len(configAndConstraints) == 0 {
		return nil, nil, nil, status.Error(codes.InvalidArgument, "nothing to parse, empty "+
			"config and constraints input")
	}
	// Standardize to JSON policies file format.
	configPolicies, err := yaml.YAMLToJSON([]byte(configAndConstraints))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error parsing config policies: %w", err)
	}
	// Extract individal config and constraints.
	var policies map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(configPolicies))
	err = dec.Decode(&policies)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error unmarshaling config policies: %s", err.Error())
	}
	var configPolicy *string
	if invariantConfig, ok := policies["invariant_config"]; ok {
		configPolicyBytes, err := json.Marshal(invariantConfig)
		if err != nil {
			return nil, nil, nil,
				fmt.Errorf("error marshaling input invariant config policy: %s", err.Error())
		}
		configPolicy = ptrs.Ptr(string(configPolicyBytes))
	}

	var constraintsPolicy *string
	if constraints, ok := policies["constraints"]; ok {
		constraintsPolicyBytes, err := json.Marshal(constraints)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error marshaling input constraints policy: %s",
				err.Error())
		}
		constraintsPolicy = ptrs.Ptr(string(constraintsPolicyBytes))
	}

	return policies, configPolicy, constraintsPolicy, nil
}

// Add or update workspace task config policies.
func (a *apiServer) PutWorkspaceConfigPolicies(
	ctx context.Context, req *apiv1.PutWorkspaceConfigPoliciesRequest,
) (*apiv1.PutWorkspaceConfigPoliciesResponse, error) {
	license.RequireLicense("manage config policies")

	// Request Validation
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	w, err := a.GetWorkspaceByID(ctx, req.WorkspaceId, *curUser, false)
	if err != nil {
		return nil, err
	}

	err = configpolicy.AuthZProvider.Get().CanModifyWorkspaceConfigPolicies(ctx, *curUser, w)
	if err != nil {
		return nil, err
	}

	err = validatePoliciesAndWorkloadType(req.WorkloadType, req.ConfigPolicies)
	if err != nil {
		return nil, err
	}

	configPolicies, invariantConfig, constraints, err := parseConfigPolicies(req.ConfigPolicies)
	if err != nil {
		return nil, err
	}

	err = configpolicy.SetTaskConfigPolicies(ctx, &model.TaskConfigPolicies{
		WorkspaceID:     ptrs.Ptr(int(req.WorkspaceId)),
		WorkloadType:    req.WorkloadType,
		LastUpdatedBy:   curUser.ID,
		LastUpdatedTime: time.Now(),
		InvariantConfig: invariantConfig,
		Constraints:     constraints,
	})
	if err != nil {
		return nil, fmt.Errorf("error setting task config policies: %w", err)
	}

	return &apiv1.PutWorkspaceConfigPoliciesResponse{
			ConfigPolicies: configpolicy.MarshalConfigPolicy(configPolicies),
		},
		err
}

// Add or update global task config policies.
func (a *apiServer) PutGlobalConfigPolicies(
	ctx context.Context, req *apiv1.PutGlobalConfigPoliciesRequest,
) (*apiv1.PutGlobalConfigPoliciesResponse, error) {
	license.RequireLicense("manage config policies")

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	err = configpolicy.AuthZProvider.Get().CanModifyGlobalConfigPolicies(ctx, curUser)
	if err != nil {
		return nil, err
	}

	err = validatePoliciesAndWorkloadType(req.WorkloadType, req.ConfigPolicies)
	if err != nil {
		return nil, err
	}

	configPolicies, invariantConfig, constraints, err := parseConfigPolicies(req.ConfigPolicies)
	if err != nil {
		return nil, err
	}

	err = configpolicy.SetTaskConfigPolicies(ctx, &model.TaskConfigPolicies{
		WorkloadType:    req.WorkloadType,
		LastUpdatedBy:   curUser.ID,
		LastUpdatedTime: time.Now(),
		InvariantConfig: invariantConfig,
		Constraints:     constraints,
	})
	if err != nil {
		return nil, fmt.Errorf("error setting task config policies: %w", err)
	}

	return &apiv1.PutGlobalConfigPoliciesResponse{
			ConfigPolicies: configpolicy.MarshalConfigPolicy(configPolicies),
		},
		err
}

// Get workspace task config policies.
func (a *apiServer) GetWorkspaceConfigPolicies(
	ctx context.Context, req *apiv1.GetWorkspaceConfigPoliciesRequest,
) (*apiv1.GetWorkspaceConfigPoliciesResponse, error) {
	license.RequireLicense("manage config policies")

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	w, err := a.GetWorkspaceByID(ctx, req.WorkspaceId, *curUser, false)
	if err != nil {
		return nil, err
	}

	err = configpolicy.AuthZProvider.Get().CanViewWorkspaceConfigPolicies(ctx, *curUser, w)
	if err != nil {
		return nil, err
	}

	resp, err := a.getConfigPolicies(ctx, ptrs.Ptr(int(req.WorkspaceId)), req.WorkloadType)
	if err != nil {
		return nil, err
	}

	return &apiv1.GetWorkspaceConfigPoliciesResponse{ConfigPolicies: resp}, nil
}

// Get global task config policies.
func (a *apiServer) GetGlobalConfigPolicies(
	ctx context.Context, req *apiv1.GetGlobalConfigPoliciesRequest,
) (*apiv1.GetGlobalConfigPoliciesResponse, error) {
	license.RequireLicense("manage config policies")

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	err = configpolicy.AuthZProvider.Get().CanViewGlobalConfigPolicies(ctx, curUser)
	if err != nil {
		return nil, err
	}

	resp, err := a.getConfigPolicies(ctx, nil, req.WorkloadType)
	if err != nil {
		return nil, err
	}

	return &apiv1.GetGlobalConfigPoliciesResponse{ConfigPolicies: resp}, nil
}

func (*apiServer) getConfigPolicies(
	ctx context.Context, workspaceID *int, workloadType string,
) (*structpb.Struct, error) {
	if !configpolicy.ValidWorkloadType(workloadType) {
		errMessage := fmt.Sprintf(invalidWorkloadTypeErr+": %s.", workloadType)
		if len(workloadType) == 0 {
			errMessage = noWorkloadErr
		}
		return nil, status.Errorf(codes.InvalidArgument, errMessage)
	}

	configPolicies, err := configpolicy.GetTaskConfigPolicies(
		ctx, workspaceID, workloadType)
	if err != nil {
		return nil, err
	}
	policyMap := map[string]interface{}{}
	if configPolicies.InvariantConfig != nil {
		var configMap map[string]interface{}
		if err := yaml.Unmarshal([]byte(*configPolicies.InvariantConfig), &configMap); err != nil {
			return nil, fmt.Errorf("unable to unmarshal json: %w", err)
		}
		policyMap["invariant_config"] = configMap
	}
	if configPolicies.Constraints != nil {
		var constraintsMap map[string]interface{}
		if err := yaml.Unmarshal([]byte(*configPolicies.Constraints), &constraintsMap); err != nil {
			return nil, fmt.Errorf("unable to unmarshal json: %w", err)
		}
		policyMap["constraints"] = constraintsMap
	}
	return configpolicy.MarshalConfigPolicy(policyMap), nil
}

// Delete workspace task config policies.
func (a *apiServer) DeleteWorkspaceConfigPolicies(
	ctx context.Context, req *apiv1.DeleteWorkspaceConfigPoliciesRequest,
) (*apiv1.DeleteWorkspaceConfigPoliciesResponse, error) {
	license.RequireLicense("manage config policies")

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	w, err := a.GetWorkspaceByID(ctx, req.WorkspaceId, *curUser, false)
	if err != nil {
		return nil, err
	}

	err = configpolicy.AuthZProvider.Get().CanModifyWorkspaceConfigPolicies(ctx, *curUser, w)
	if err != nil {
		return nil, err
	}

	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		errMessage := fmt.Sprintf(invalidWorkloadTypeErr+": %s.", req.WorkloadType)
		if len(req.WorkloadType) == 0 {
			errMessage = noWorkloadErr
		}
		return nil, status.Errorf(codes.InvalidArgument, errMessage)
	}

	err = configpolicy.DeleteConfigPolicies(ctx, ptrs.Ptr(int(req.WorkspaceId)),
		req.WorkloadType)
	if err != nil {
		return nil, err
	}
	return &apiv1.DeleteWorkspaceConfigPoliciesResponse{}, nil
}

// Delete global task config policies.
func (a *apiServer) DeleteGlobalConfigPolicies(
	ctx context.Context, req *apiv1.DeleteGlobalConfigPoliciesRequest,
) (*apiv1.DeleteGlobalConfigPoliciesResponse, error) {
	license.RequireLicense("manage config policies")

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	err = configpolicy.AuthZProvider.Get().CanModifyGlobalConfigPolicies(ctx, curUser)
	if err != nil {
		return nil, err
	}

	if !configpolicy.ValidWorkloadType(req.WorkloadType) {
		errMessage := fmt.Sprintf(invalidWorkloadTypeErr+": %s.", req.WorkloadType)
		if len(req.WorkloadType) == 0 {
			errMessage = noWorkloadErr
		}
		return nil, status.Errorf(codes.InvalidArgument, errMessage)
	}

	err = configpolicy.DeleteConfigPolicies(ctx, nil, req.WorkloadType)
	if err != nil {
		return nil, err
	}
	return &apiv1.DeleteGlobalConfigPoliciesResponse{}, nil
}

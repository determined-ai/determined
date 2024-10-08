package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
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
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	noWorkloadErr          = "no workload type specified."
	noPoliciesErr          = "no specified config policies."
	invalidWorkloadTypeErr = "invalid workload type"
)

func (a *apiServer) validatePoliciesAndWorkloadType(workloadType, configPolicies string) error {
	ctx := context.Background() // TODO CAROLINA
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
	var expConfigPolicies *expconf.ExperimentConfigV0
	var ntscConfigPolicies *model.CommandConfig
	var constraints *model.Constraints
	if workloadType == model.ExperimentType {
		cp, err := configpolicy.UnmarshalExperimentConfigPolicy(configPolicies)
		if err != nil {
			return err
		}
		expConfigPolicies = cp.InvariantConfig
		constraints = cp.Constraints
	} else {
		cp, err := configpolicy.UnmarshalNTSCConfigPolicy(configPolicies)
		if err != nil {
			return err
		}
		ntscConfigPolicies = cp.InvariantConfig
		constraints = cp.Constraints
	}

	// Now validate against global priority, max slots, and other fields.
	if constraints != nil {
		if err := a.checkAgainstGlobalPriority(ctx, constraints.PriorityLimit, workloadType); err != nil {
			return err
		}
	}

	if expConfigPolicies != nil {
		if err := a.checkAgainstGlobalPriority(ctx, expConfigPolicies.Resources().Priority(), workloadType); err != nil {
			return err
		}
		if err := a.checkAgainstGlobalConfig(ctx, expConfigPolicies, nil, workloadType); err != nil {
			return err
		}
		if err := a.checkConstraintConflicts(
			constraints, *expConfigPolicies.RawResources.RawMaxSlots,
			*expConfigPolicies.RawResources.RawSlotsPerTrial, *expConfigPolicies.RawResources.RawPriority,
		); err != nil {
			return err
		}
	}

	if ntscConfigPolicies != nil {
		if err := a.checkAgainstGlobalPriority(ctx, ntscConfigPolicies.Resources.Priority, workloadType); err != nil {
			return err
		}
		if err := a.checkAgainstGlobalConfig(ctx, nil, ntscConfigPolicies, workloadType); err != nil {
			return err
		}
		if err := a.checkConstraintConflicts(
			constraints, *ntscConfigPolicies.Resources.MaxSlots,
			ntscConfigPolicies.Resources.Slots, *ntscConfigPolicies.Resources.Priority,
		); err != nil {
			return err
		}
	}
	return nil
}

func (a *apiServer) checkAgainstGlobalPriority(ctx context.Context, taskPriority *int, workloadType string) error {
	if taskPriority != nil {
		_, priorityEnabledErr := a.m.rm.SmallerValueIsHigherPriority()
		if priorityEnabledErr != nil {
			return fmt.Errorf("task priority is not supported in this cluster: %w", priorityEnabledErr)
		}
		_, globalPriorityExists, _ := configpolicy.GetPriorityLimit(ctx, nil, workloadType)
		if globalPriorityExists {
			return fmt.Errorf("global priority limit already exists for the task config policy")
		}
	}
	return nil
}

func (a *apiServer) checkConstraintConflicts(constraints *model.Constraints, maxSlots, slots, priority int) error {
	if constraints != nil {
		if *constraints.PriorityLimit != priority {
			return fmt.Errorf("invariant config & constraints are trying to set the priority limit")
		}
		if *constraints.ResourceConstraints.MaxSlots != maxSlots {
			return fmt.Errorf("invariant config & constraints are trying to set the max slots")
		}
		if *constraints.ResourceConstraints.MaxSlots > slots {
			return fmt.Errorf("invariant config & constraints are attempting to set an invalid max slot")
		}
	}
	return nil
}

func (a *apiServer) checkAgainstGlobalConfig(
	ctx context.Context, expConfig *expconf.ExperimentConfigV0, ntscConfig *model.CommandConfig, workloadType string,
) error {
	globalConfigPolicies, err := configpolicy.GetTaskConfigPolicies(ctx, nil, workloadType)
	if err != nil {
		return fmt.Errorf("error in getting global scope task config policy: %w", err)
	}

	globalNTSCConfig, err := configpolicy.UnmarshalNTSCConfigPolicy(*globalConfigPolicies.InvariantConfig)
	if err != nil {
		return err
	}
	if err = haveEqualDefinedFields(globalNTSCConfig.InvariantConfig, ntscConfig); err != nil {
		return err
	}

	globalExpConfig, err := configpolicy.UnmarshalExperimentConfigPolicy(*globalConfigPolicies.InvariantConfig)
	if err != nil {
		return err
	}
	if err = haveEqualDefinedFields(globalExpConfig.InvariantConfig, expConfig); err != nil {
		return err
	}

	return nil
}

func haveEqualDefinedFields(config1, config2 interface{}) error {
	v1 := reflect.ValueOf(config1)
	v2 := reflect.ValueOf(config2)

	// If the values are pointers, dereference them
	if v1.Kind() == reflect.Ptr {
		v1 = v1.Elem()
	}
	if v2.Kind() == reflect.Ptr {
		v2 = v2.Elem()
	}

	// Check if both values are valid structs
	if v1.Kind() != reflect.Struct || v2.Kind() != reflect.Struct {
		return fmt.Errorf("both inputs must be structs")
	}

	hasSharedField := false

	// Iterate over the fields in the struct
	for i := 0; i < v1.NumField(); i++ {
		field1 := v1.Field(i)
		field2 := v2.Field(i)

		// Check if the field is a pointer, map, or interface (which can be nil)
		if field1.Kind() == reflect.Ptr || field1.Kind() == reflect.Map || field1.Kind() == reflect.Interface {
			if !field1.IsNil() && !field2.IsNil() {
				hasSharedField = true
				// Compare the dereferenced values
				if !reflect.DeepEqual(field1.Interface(), field2.Interface()) {
					return fmt.Errorf("shared non-null field has different values")
				}
			}
		} else if field1.IsValid() && field2.IsValid() && !field1.IsZero() && !field2.IsZero() {
			hasSharedField = true
			// For non-pointer fields, compare directly if both are non-zero
			if !reflect.DeepEqual(field1.Interface(), field2.Interface()) {
				return fmt.Errorf("shared non-null field has different values")
			}
		}
	}

	if !hasSharedField {
		return nil // No shared non-null fields to compare
	}

	return nil // Configs are equal in shared non-null fields
}

func parseConfigPolicies(configAndConstraints string) (
	tcps map[string]interface{}, invariantConfig *string, constraints *string, err error,
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
	// Extract individual config and constraints.
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

	err = a.validatePoliciesAndWorkloadType(req.WorkloadType, req.ConfigPolicies)
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

	err = a.validatePoliciesAndWorkloadType(req.WorkloadType, req.ConfigPolicies)
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

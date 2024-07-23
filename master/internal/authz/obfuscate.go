package authz

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

const (
	hiddenString = "********"
	hiddenInt    = -1
)

// ObfuscateDevice obfuscates sensitive information in given Device.
func ObfuscateDevice(device *devicev1.Device) error {
	if device == nil {
		return errors.New("device must be defined")
	}
	device.Id = hiddenInt
	device.Uuid = hiddenString
	return nil
}

// ObfuscateContainer obfuscates sensitive information in given Container.
func ObfuscateContainer(container *containerv1.Container) error {
	if container == nil {
		return errors.New("container must be defined")
	}
	container.Id = hiddenString
	container.Parent = hiddenString
	container.PermissionDenied = true
	for _, device := range container.Devices {
		if err := ObfuscateDevice(device); err != nil {
			return err
		}
	}
	return nil
}

// ObfuscateSlot obfuscates sensitive information in given Slot.
func ObfuscateSlot(slot *agentv1.Slot) error {
	if slot == nil {
		return errors.New("slot must be defined")
	}
	if err := ObfuscateDevice(slot.Device); err != nil {
		return errors.Errorf("unable to obfuscate slot: %s", err)
	}
	if slot.Container != nil {
		if err := ObfuscateContainer(slot.Container); err != nil {
			return errors.Errorf("unable obfuscate slot: %s", err)
		}
	}
	return nil
}

// ObfuscateAgent obfuscates sensitive information in given Agent.
func ObfuscateAgent(agent *agentv1.Agent) error {
	if agent == nil {
		return errors.New("agent must be defined")
	}
	agent.Addresses = []string{hiddenString}

	if agent.Containers != nil {
		obfuscatedContainers := make(map[string]*containerv1.Container)
		for _, container := range agent.Containers {
			obfuscatedKey := uuid.New().String()
			obfuscatedContainers[obfuscatedKey] = container
		}
		agent.Containers = obfuscatedContainers
		for _, container := range agent.Containers {
			if err := ObfuscateContainer(container); err != nil {
				return errors.Errorf("unable to obfuscate agent: %s", err)
			}
		}
	}

	// Retain map lexicographically order so the webui doesn't hop around every refresh.
	slotIDToObfuscated := make(map[string]*agentv1.Slot)
	var slotIDs []string
	for _, slot := range agent.Slots {
		if err := ObfuscateSlot(slot); err != nil {
			return errors.Errorf("unable to obfuscate agent: %s", err)
		}

		slotIDToObfuscated[slot.Id] = slot
		slotIDs = append(slotIDs, slot.Id)
	}
	slices.Sort(slotIDs)
	obfuscatedSlots := make(map[string]*agentv1.Slot)
	for i, slotID := range slotIDs {
		s := slotIDToObfuscated[slotID]
		s.Id = model.SortableSlotIndex(i)
		obfuscatedSlots[s.Id] = s
	}
	agent.Slots = obfuscatedSlots

	return nil
}

// ObfuscateJob obfuscates sensitive information in given Job.
func ObfuscateJob(job *jobv1.Job) jobv1.LimitedJob {
	return jobv1.LimitedJob{
		Summary:        job.Summary,
		Type:           job.Type,
		ResourcePool:   job.ResourcePool,
		IsPreemptible:  job.IsPreemptible,
		Priority:       job.Priority,
		Weight:         job.Weight,
		JobId:          job.JobId,
		RequestedSlots: job.RequestedSlots,
		AllocatedSlots: job.AllocatedSlots,
		Progress:       job.Progress,
		WorkspaceId:    job.WorkspaceId,
	}
}

// ObfuscateExperiments obfuscates sensitive information in experiments.
// Currently, that is considered to be anything the user has configured
// under a "secrets" key in the general-purpose "data" config.
func ObfuscateExperiments(experiments ...*experimentv1.Experiment) error {
	for _, exp := range experiments {
		data, exists := exp.Config.Fields["data"] //nolint:staticcheck
		if !exists {
			continue
		}
		dataMap := data.GetStructValue() // nil if not struct
		if dataMap == nil {
			continue
		}
		secrets, exists := dataMap.Fields["secrets"]
		if !exists {
			continue
		}
		secretsMap := secrets.GetStructValue()
		if secretsMap == nil {
			continue
		}
		for key := range secretsMap.Fields {
			secretsMap.Fields[key] = structpb.NewStringValue(hiddenString)
		}

		var oConfig map[string]interface{}
		err := json.Unmarshal([]byte(exp.OriginalConfig), &oConfig)
		if err != nil {
			err = yaml.Unmarshal([]byte(exp.OriginalConfig), &oConfig)
			if err != nil {
				return fmt.Errorf("error unmarshaling original experiment config: %w", err)
			}
		}
		oData, exists := oConfig["data"]
		if !exists {
			continue
		}
		oDataMap, ok := oData.(map[string]interface{})
		if !ok {
			continue
		}
		oSecrets, exists := oDataMap["secrets"]
		if !exists {
			continue
		}
		oSecretsMap, ok := oSecrets.(map[string]interface{})
		if !ok {
			continue
		}
		for key := range oSecretsMap {
			oSecretsMap[key] = hiddenString
		}
		pConfig, err := json.Marshal(oConfig)
		if err != nil {
			return fmt.Errorf("error remarshaling experiment config: %w", err)
		}
		exp.OriginalConfig = string(pConfig)
	}
	return nil
}

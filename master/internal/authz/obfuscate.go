package authz

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

const (
	HiddenString = "********"
	HiddenInt    = -1
	HiddenBool   = false
)

// ObfuscateDevice obfuscates sensitive information in given Device.
func ObfuscateDevice(device *devicev1.Device) error {
	if device == nil {
		return errors.New("device must be defined")
	}
	device.Id = HiddenInt
	device.Uuid = HiddenString
	return nil
}

// ObfuscateContainer obfuscates sensitive information in given Container.
func ObfuscateContainer(container *containerv1.Container) error {
	if container == nil {
		return errors.New("container must be defined")
	}
	container.Id = HiddenString
	container.Parent = HiddenString
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
	agent.Addresses = []string{HiddenString}
	agent.Id = ""

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

	obfuscatedSlots := make(map[string]*agentv1.Slot)
	for _, slot := range agent.Slots {
		if err := ObfuscateSlot(slot); err != nil {
			return errors.Errorf("unable to obfuscate agent: %s", err)
		}
		obfuscatedKey := uuid.New().String()
		obfuscatedSlots[obfuscatedKey] = slot
	}
	agent.Slots = obfuscatedSlots

	return nil
}

// ObfuscateNTSCTask obfuscates sensitive information about a given Notebook, Tensorboard, Shell,
// or Command Task.
func ObfuscateNTSCTask(ntscTask sproto.AllocationSummary) (sproto.AllocationSummary, error) {
	obfuscatedNTSCTask := sproto.AllocationSummary{}

	obfuscatedNTSCTask.TaskID = HiddenString
	obfuscatedNTSCTask.AllocationID = HiddenString
	obfuscatedNTSCTask.Name = ""

	resSummary := sproto.ResourcesSummary{}

	for _, r := range ntscTask.Resources {
		resSummary.ResourcesID = HiddenString
		resSummary.AllocationID = HiddenString
		var cID cproto.ID = HiddenString
		resSummary.ContainerID = &cID
		resourceAgentDevices := make(map[aproto.ID][]device.Device)
		var obfuscatedDevs []device.Device
		for _, devs := range r.AgentDevices {
			for _, dev := range devs {
				dev.Brand = HiddenString
				dev.UUID = HiddenString
				obfuscatedDevs = append(obfuscatedDevs, dev)
			}
			resourceAgentDevices[aproto.ID(HiddenString)] = obfuscatedDevs
		}
		resSummary.AgentDevices = resourceAgentDevices
		obfuscatedNTSCTask.Resources = append(obfuscatedNTSCTask.Resources, resSummary)
	}
	obfuscatedProxyPorts := make([]*sproto.ProxyPortConfig, len(ntscTask.ProxyPorts))
	for i := range ntscTask.ProxyPorts {
		ppConf := sproto.ProxyPortConfig{}
		ppConf.ServiceID = HiddenString
		ppConf.Port = HiddenInt
		ppConf.ProxyTCP = HiddenBool
		ppConf.Unauthenticated = HiddenBool
		obfuscatedProxyPorts[i] = &ppConf
	}
	obfuscatedNTSCTask.ProxyPorts = obfuscatedProxyPorts
	return obfuscatedNTSCTask, nil
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

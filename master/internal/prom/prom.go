package prom

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/determined-ai/determined/master/internal/sproto"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"
)

var (
	containerIDToAllocationID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "container_id_allocation_id",
		Help: `
Exposes mapping of allocation ID to container ID`,
	}, []string{"container_id", "allocation_id"})

	allocationIDToTaskID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "allocation_id_task_id",
		Help: `
Exposes mapping of allocation ID to task ID`,
	}, []string{"allocation_id", "task_id"})

	containerIDToRuntimeID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "container_id_runtime_container_id",
		Help:      "a mapping of the container ID to the container ID given be the runtime",
	}, []string{"container_runtime_id", "container_id"})


	experimentIDToLabels = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "experiment_id_label",
		Help:      "a mapping of the experiment ID to the labels",
	}, []string{"experiment_id", "label"})

	gpuUUIDToContainerID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "gpu_uuid_container_id",
		Help: `
Exposes mapping of Determined's internal container ID to the GPU UUID/device ID as given by nvidia-smi
`,
	}, []string{"gpu_uuid", "container_id"})

	// Reg is a prometheus registry containing all exported user-facing metrics.
	DetStateMetrics = prometheus.NewRegistry()
)

const (
	cAdvisorPort = ":8080"
	dcgmPort     = ":9400"

	// The are extra labels added to metrics on scrape.
	detAgentIDLabel      = "det_agent_id"
	detResourcePoolLabel = "det_resource_pool"

	targetsFile = "/etc/determined/targets.json"
)

type fileSDConfigEntry struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

func init() {
	DetStateMetrics.MustRegister(containerIDToAllocationID)
	DetStateMetrics.MustRegister(containerIDToRuntimeID)
	DetStateMetrics.MustRegister(gpuUUIDToContainerID)
	DetStateMetrics.MustRegister(experimentIDToLabels)
	DetStateMetrics.MustRegister(allocationIDToTaskID)
}

func AssociateAllocationContainer(aID string, cID string) {
	containerIDToAllocationID.WithLabelValues(cID, aID).Inc()
}

func AssociateAllocationTask(aID string, tID string) {
	allocationIDToTaskID.WithLabelValues(aID, tID).Inc()
}

func DisassociateAllocationTask(aID string, tID string) {
	allocationIDToTaskID.WithLabelValues(aID, tID).Dec()
}

func AssociateContainerRuntimeID(cID string, dcID string) {
	containerIDToRuntimeID.WithLabelValues(dcID, cID).Inc()
}

func AddAllocationContainer(summary sproto.ContainerSummary) {
	AssociateAllocationContainer(summary.AllocationID.String(), summary.ID.String())
	for _, d := range summary.Devices {
		AssociateContainerGPU(summary.ID.String(), d)
	}
}

func RemoveAllocationContainer(summary sproto.ContainerSummary) {
	DisassociateAllocationContainer(summary.AllocationID.String(), summary.ID.String())
	for _, d := range summary.Devices {
		DisassociateContainerGPU(summary.ID.String(), d)
	}
}

func DisassociateAllocationContainer(aID string, cID string) {
	containerIDToAllocationID.WithLabelValues(cID, aID).Dec()
}

func AssociateExperimentIDLabels(eID string, labels []string) {
	for i := range labels {
		experimentIDToLabels.WithLabelValues(eID, labels[i]).Inc()
	}
}

func AssociateContainerGPU(cID string, d device.Device) {
	if d.Type == device.GPU {
		gpuUUIDToContainerID.
			WithLabelValues(d.UUID, cID).
			Inc()
	}
}

func DisassociateContainerGPU(cID string, d device.Device) {
	if d.Type != device.GPU {
		return
	}

	gpuUUIDToContainerID.WithLabelValues(d.UUID, cID).Dec()
	gpuUUIDToContainerID.DeleteLabelValues(d.UUID, cID)
}

// AddAgentAsTarget adds an entry to a list of currently active agents in a target JSON-formatted file
// This file is watched by Prometheus for changes to which targets to scrape
func AddAgentAsTarget(
	ctx *actor.Context,
	agentId string,
	agentAddress string,
	agentResourcePool string) {
	ctx.Log().Infof("Adding agent %s at %s as a prometheus target", agentId, agentAddress)

	if _, err := os.Stat(targetsFile); os.IsNotExist(err) {
		pwd, err := os.Getwd()
		if err == nil {
			ctx.Log().Infof("pwd %v", pwd)
		}
		_, err = os.OpenFile(targetsFile, os.O_RDONLY|os.O_CREATE, 0666)
		if err != nil {
			ctx.Log().Errorf("Error creating targets config file %s", err)
		}
	}

	fileSDConfig := fileSDConfigEntry{
		Targets: []string{
			agentAddress + dcgmPort,
			agentAddress + cAdvisorPort,
		}, Labels: map[string]string{
			detAgentIDLabel:      agentId,
			detResourcePoolLabel: agentResourcePool,
		},
	}

	fileSDConfigs := getFileSDConfigs()

	fileSDConfigs = append(fileSDConfigs, fileSDConfig)

	err := writeConfigsToTargetsFile(fileSDConfigs)

	if err != nil {
		ctx.Log().Errorf("Error adding entry to file %s", err)
	}
}

// RemoveAgentAsTarget removes agent from the file SD targets config used by prometheus
// to scrape agent addresses
func RemoveAgentAsTarget(ctx *actor.Context,
	agentId string,
) {
	ctx.Log().Infof("Removing %s as a target", agentId)

	fileSDConfigs := getFileSDConfigs()

	for i := range fileSDConfigs {
		ctx.Log().Infof("Checking %s", fileSDConfigs[i].Labels[detAgentIDLabel])

		if fileSDConfigs[i].Labels[detAgentIDLabel] == agentId {
			ctx.Log().Infof("Removing agent %s from targets.json", agentId)
			fileSDConfigs = append(fileSDConfigs[:i], fileSDConfigs[i+1:]...)
			break
		}
	}

	err := writeConfigsToTargetsFile(fileSDConfigs)

	if err != nil {
		ctx.Log().Errorf("Error updating targets file %s", err)
	}
}

func getFileSDConfigs() []fileSDConfigEntry {
	var fileSDConfigs []fileSDConfigEntry

	fileContent, _ := ioutil.ReadFile(targetsFile)

	_ = json.Unmarshal(fileContent, &fileSDConfigs)

	return fileSDConfigs
}

func writeConfigsToTargetsFile(configs []fileSDConfigEntry) error {

	targetsJson, err := json.MarshalIndent(configs, "", "  ")

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(targetsFile, targetsJson, 0644) //nolint: gosec

	if err != nil {
		return err
	}

	return nil
}

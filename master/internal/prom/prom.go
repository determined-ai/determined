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
	// Gauge that maps tasks to their container IDs.
	containerIDToAllocationID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "container_id_allocation_id",
		Help: `
Exposes mapping of container ID to allocation ID.

Task ID is the ID of the task within determined. This can be a little opaque but is shown
by 'det task list', which also provides a mapping to a more human-readable task name.

Container ID is Determined's internal identifier for a container or pod and appears
as a label on containers and metadata on pods (and thus in labels collected by monitoring tools).
This is useful to join in on metrics from those monitoring tools (e.g. cAdvisor).
`,
	}, []string{"container_id", "allocation_id"})

	containerIDToRuntimeID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "container_id_runtime_container_id",
		Help:      "a mapping of the container ID to the container ID given be the runtime",
	}, []string{"container_runtime_id", "container_id"})

	taskActorToAllocation = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "allocation_id_task_actor",
		Help:      "a mapping of the task ID to the task actor initiating it",
	}, []string{"allocation_id", "task_actor"})

	containerIDToExperimentID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "container_id_experiment_id",
		Help:      "a mapping of the container ID to the experiment and trial",
	}, []string{"container_id", "experiment_id", "trial_id"})

	experimentIDToLabels = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "experiment_id_label",
		Help:      "a mapping of the experiment ID to the labels",
	}, []string{"experiment_id", "label"})

	gpuUUIDToContainerID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "gpu_uuid_container_id",
		Help: `
Exposes mapping of task name to container ID to GPU uuid.

Container ID is Determined's internal identifier for a container or pod and appears
as a label on containers or pods (and thus in container or pod monitoring tools) as
"ai.determined.container_id". This is useful to join in on metrics from those monitoring
tools (e.g. cAdvisor).

GPU UUID is the device ID as given by NVML (or nvidia-smi). This is useful to join in on
GPU metrics from other monitoring tools (e.g. DCGM).
`,
	}, []string{"gpu_uuid", "container_id"})

	// Reg is a prometheus registry containing all exported user-facing metrics.
	DetStateMetrics = prometheus.NewRegistry()
)

const (
	cAdvisorExporter = ":8080"
	dcgmExporter     = ":9400"

	// The are extra labels added to metrics on scrape.
	detAgentIDLabel      = "det_agent_id"
	detResourcePoolLabel = "det_resource_pool"
	detAgentName         = "det_label"

	targetsFile = "./prometheus/targets.json"
)

type fileSDConfigEntry struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

func init() {
	DetStateMetrics.MustRegister(containerIDToAllocationID)
	DetStateMetrics.MustRegister(containerIDToRuntimeID)
	DetStateMetrics.MustRegister(gpuUUIDToContainerID)
	DetStateMetrics.MustRegister(taskActorToAllocation)
	DetStateMetrics.MustRegister(containerIDToExperimentID)
	DetStateMetrics.MustRegister(experimentIDToLabels)
}

// AllocationContainer records the given task owns the given container.
func AllocationContainer(cID string, aID string) {
	containerIDToAllocationID.WithLabelValues(cID, aID).Inc()
}

// AddAllocation records the given allocation
func AddAllocation(summary sproto.ContainerSummary) {
	cID := summary.ID.String()
	aID := summary.AllocationID.String()
	containerIDToAllocationID.WithLabelValues(cID, aID).Inc()
	AssociateContainerGPUs(cID, summary.Devices...)
}

// DisassociateTaskContainer records the given task no longer owns the given container.
func DisassociateTaskContainer(tID string, cID string) {
	containerIDToAllocationID.WithLabelValues(cID, tID).Dec()
}

func AssociateTaskActor(actor string, aID string) {
	taskActorToAllocation.WithLabelValues(actor, aID).Inc()
}

// DisassociateTaskActor records the given task owns the given container.
func DisassociateTaskActor(tID string, actor string) {
	taskActorToAllocation.WithLabelValues(tID, actor).Dec()
}

// AllocationContainer associated the given Determined container ID with a runtime ID (e.g. Docker ID).
func AssociateContainerRuntimeID(cID string, dcID string) {
	containerIDToRuntimeID.WithLabelValues(dcID, cID).Inc()
}

func AssociateContainerExperimentID(cID string, eID string, tID string) {
	experimentIDToLabels.WithLabelValues(eID, "").Inc()
	containerIDToExperimentID.WithLabelValues(cID, eID, tID).Inc()
}

func AssociateExperimentIDLabels(eID string, labels []string) {
	for i := range labels {
		experimentIDToLabels.WithLabelValues(eID, labels[i]).Inc()
	}
}

func DisassociateContainerExperimentID(cID string, eID string, tID string) {
	containerIDToExperimentID.WithLabelValues(cID, eID, tID).Dec()
}

// DisassociateTaskContainer records the given task no longer owns the given container.
func DisassociateContainerRuntimeID(cID string, dcID string) {
	containerIDToAllocationID.WithLabelValues(cID, dcID).Dec()
}

// AssociateContainerGPUs records a usage of some devices by the specified a Determined container.
func AssociateContainerGPUs(cID string, ds ...device.Device) {
	for _, d := range ds {
		if d.Type == device.GPU {
			gpuUUIDToContainerID.
				WithLabelValues(d.UUID, cID).
				Inc()
		}
	}
}

// DisassociateContainerGPUs records a completion of usage of some devices by the specified a Determined container.
func DisassociateContainerGPUs(cID string, ds ...device.Device) {
	for _, d := range ds {
		if d.Type == device.GPU {
			gpuUUIDToContainerID.
				WithLabelValues(d.UUID, cID).
				Dec()
			//Need to Delete after prom has scraped the 0.
			//gpuUUIDToContainerID.
			//	// Note, theses labels are order-sensitive. Out of order is a memory leak.
			//	DeleteLabelValues(d.UUID, cID)
		}
	}
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
			agentAddress + dcgmExporter,
			agentAddress + cAdvisorExporter,
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

	err = ioutil.WriteFile(targetsFile, targetsJson, 0644)

	if err != nil {
		return err
	}

	return nil
}

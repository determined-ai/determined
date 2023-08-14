package prom

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/determined-ai/determined/master/pkg/schemas/expconf"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/pkg/device"
)

var (
	containerIDToAllocationID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "container_id_allocation_id",
		Help: `
Exposes mapping of allocation ID to container ID`,
	}, []string{"container_id", "allocation_id"})

	allocationIDToTask = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "allocation_id_task_id_task_actor",
		Help: `
Exposes mapping of allocation ID to task ID and actor`,
	}, []string{"allocation_id", "task_id", "task_actor", "job_id"})

	containerIDToRuntimeID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "container_id_runtime_container_id",
		Help:      "a mapping of the container ID to the container ID given be the runtime",
	}, []string{"container_runtime_id", "container_id"})

	jobIDToExperimentID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "job_id_experiment_id",
		Help:      "a mapping of the job ID to the experiment ID",
	}, []string{"job_id", "experiment_id"})

	experimentIDToLabels = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "experiment_id_label",
		Help:      "a mapping of the experiment ID to the labels",
	}, []string{"experiment_id", "label"})

	gpuUUIDToContainerID = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "det",
		Name:      "gpu_uuid_container_id",
		Help: `
Exposes mapping of Determined's container ID to the GPU UUID/device ID as given by nvidia-smi
`,
	}, []string{"gpu_uuid", "container_id"})

	// DetStateMetrics is a prometheus registry containing all exported user-facing metrics.
	DetStateMetrics = prometheus.NewRegistry()
)

const (
	// CAdvisorPort is the default port for cAdvisor.
	CAdvisorPort = ":8080"

	// DcgmPort is the default port for DCGM.
	DcgmPort = ":9400"

	// DetAgentIDLabel is the internal ID for the Determined agent.
	DetAgentIDLabel = "det_agent_id"

	// DetResourcePoolLabel is the resource pool name.
	DetResourcePoolLabel = "det_resource_pool"
)

// TargetSDConfig is the format for specifying targets for prometheus service discovery.
type TargetSDConfig struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

func init() { //nolint: gochecknoinits
	DetStateMetrics.MustRegister(containerIDToAllocationID)
	DetStateMetrics.MustRegister(containerIDToRuntimeID)
	DetStateMetrics.MustRegister(gpuUUIDToContainerID)
	DetStateMetrics.MustRegister(experimentIDToLabels)
	DetStateMetrics.MustRegister(allocationIDToTask)
	DetStateMetrics.MustRegister(jobIDToExperimentID)
}

// AssociateAllocationContainer associates an allocation with its container ID.
func AssociateAllocationContainer(aID model.AllocationID, cID cproto.ID) {
	containerIDToAllocationID.WithLabelValues(cID.String(), aID.String()).Inc()
}

// AssociateAllocationTask associates an allocation ID with its task/job info.
func AssociateAllocationTask(aID model.AllocationID,
	tID model.TaskID,
	name string,
	jID model.JobID,
) {
	allocationIDToTask.WithLabelValues(aID.String(), tID.String(), name, jID.String()).Inc()
}

// AssociateJobExperiment associates a job ID with experiment info.
func AssociateJobExperiment(jID model.JobID, eID string, labels expconf.Labels) {
	jobIDToExperimentID.WithLabelValues(jID.String(), eID).Inc()
	var expLabels []string

	for l := range labels {
		expLabels = append(expLabels, l)
	}

	AssociateExperimentIDLabels(eID, expLabels)
}

// DisassociateJobExperiment disassociates a job ID with experiment info.
func DisassociateJobExperiment(jID model.JobID, eID string, labels expconf.Labels) {
	jobIDToExperimentID.WithLabelValues(jID.String(), eID).Dec()
	expLabels := make([]string, len(labels))

	for l := range labels {
		expLabels = append(expLabels, l)
	}

	DisassociateExperimentIDLabels(eID, expLabels)
}

// DisassociateAllocationTask disassociates an allocation ID with its task info.
func DisassociateAllocationTask(aID model.AllocationID, tID model.TaskID, name string,
	jID model.JobID,
) {
	allocationIDToTask.WithLabelValues(aID.String(), tID.String(), name, jID.String()).Dec()
}

// AssociateContainerRuntimeID associates a Determined container ID with the runtime container ID.
func AssociateContainerRuntimeID(cID cproto.ID, dcID string) {
	containerIDToRuntimeID.WithLabelValues(dcID, cID.String()).Inc()
}

// AddAllocationResources associates allocation and container and container and GPUs.
func AddAllocationResources(summary sproto.ResourcesSummary,
	containerStarted *sproto.ResourcesStarted,
) {
	if summary.ContainerID == nil {
		return
	}

	AssociateAllocationContainer(summary.AllocationID, *summary.ContainerID)
	AssociateContainerRuntimeID(*summary.ContainerID, containerStarted.NativeResourcesID)
	for _, ds := range summary.AgentDevices {
		for _, d := range ds {
			AssociateContainerGPU(*summary.ContainerID, d)
		}
	}
}

// RemoveAllocationResources disassociates allocation and container and container and its GPUs.
func RemoveAllocationResources(summary sproto.ResourcesSummary) {
	if summary.ContainerID == nil {
		return
	}

	DisassociateAllocationContainer(summary.AllocationID, *summary.ContainerID)
	for _, ds := range summary.AgentDevices {
		for _, d := range ds {
			DisassociateContainerGPU(*summary.ContainerID, d)
		}
	}
}

// DisassociateAllocationContainer disassociates allocation ID with its container ID.
func DisassociateAllocationContainer(aID model.AllocationID, cID cproto.ID) {
	containerIDToAllocationID.WithLabelValues(cID.String(), aID.String()).Dec()
}

// AssociateExperimentIDLabels associates experiment ID with a list of labels.
func AssociateExperimentIDLabels(eID string, labels []string) {
	for i := range labels {
		experimentIDToLabels.WithLabelValues(eID, labels[i]).Inc()
	}
}

// DisassociateExperimentIDLabels disassociates experiment ID with a list of labels.
func DisassociateExperimentIDLabels(eID string, labels []string) {
	for i := range labels {
		experimentIDToLabels.WithLabelValues(eID, labels[i]).Dec()
	}
}

// AssociateContainerGPU associates container ID with GPU device ID.
func AssociateContainerGPU(cID cproto.ID, d device.Device) {
	if d.Type == device.CUDA {
		gpuUUIDToContainerID.
			WithLabelValues(d.UUID, cID.String()).
			Inc()
	}
}

// DisassociateContainerGPU removes association between container ID and device ID.
func DisassociateContainerGPU(cID cproto.ID, d device.Device) {
	if d.Type != device.CUDA {
		return
	}

	gpuUUIDToContainerID.WithLabelValues(d.UUID, cID.String()).Dec()
	gpuUUIDToContainerID.DeleteLabelValues(d.UUID, cID.String())
}

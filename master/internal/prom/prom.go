package prom

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
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
	}, []string{"allocation_id", "task_id", "task_actor"})

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
Exposes mapping of Determined's container ID to the GPU UUID/device ID as given by nvidia-smi
`,
	}, []string{"gpu_uuid", "container_id"})

	// DetStateMetrics is a prometheus registry containing all exported user-facing metrics.
	DetStateMetrics = prometheus.NewRegistry()
)

const (
	// CAdvisorPort is the default port for cAdvisor
	CAdvisorPort = ":8080"

	// DcgmPort is the default port for DCGM
	DcgmPort = ":9400"

	// DetAgentIDLabel is the internal ID for the Determined agent
	DetAgentIDLabel = "det_agent_id"

	// DetResourcePoolLabel is the resource pool name
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
}

// AssociateAllocationContainer associates an allocation with its container ID.
func AssociateAllocationContainer(aID string, cID string) {
	containerIDToAllocationID.WithLabelValues(cID, aID).Inc()
}

// AssociateAllocationTask associates an allocation ID with its task info.
func AssociateAllocationTask(aID model.AllocationID,
	tID model.TaskID,
	taskActor actor.Address) { //nolint: interfacer
	allocationIDToTask.WithLabelValues(aID.String(), tID.String(), taskActor.String()).Inc()
}

// DisassociateAllocationTask disassociates an allocation ID with its task info.
func DisassociateAllocationTask(aID string, tID string, taskActor string) {
	allocationIDToTask.WithLabelValues(aID, tID, taskActor).Dec()
}

// AssociateContainerRuntimeID associates a Determined container ID with the runtime container ID.
func AssociateContainerRuntimeID(cID string, dcID string) {
	containerIDToRuntimeID.WithLabelValues(dcID, cID).Inc()
}

// AddAllocationReservation associates allocation and container and container and GPUs.
func AddAllocationReservation(summary sproto.ContainerSummary,
	containerStarted *sproto.TaskContainerStarted) {
	AssociateAllocationContainer(summary.AllocationID.String(), summary.ID.String())
	AssociateContainerRuntimeID(summary.ID.String(), containerStarted.NativeReservationID)
	for _, d := range summary.Devices {
		AssociateContainerGPU(summary.ID, d)
	}
}

// RemoveAllocationReservation disassociates allocation and container and container and its GPUs.
func RemoveAllocationReservation(summary sproto.ContainerSummary) {
	DisassociateAllocationContainer(summary.AllocationID.String(), summary.ID.String())
	for _, d := range summary.Devices {
		DisassociateContainerGPU(summary.ID.String(), d)
	}
}

// DisassociateAllocationContainer disassociates allocation ID with its container ID.
func DisassociateAllocationContainer(aID string, cID string) {
	containerIDToAllocationID.WithLabelValues(cID, aID).Dec()
}

// AssociateExperimentIDLabels assicates experiment ID with a list of labels.
func AssociateExperimentIDLabels(eID string, labels []string) {
	for i := range labels {
		experimentIDToLabels.WithLabelValues(eID, labels[i]).Inc()
	}
}

// AssociateContainerGPU associates container ID with GPU device ID.
func AssociateContainerGPU(cID cproto.ID, d device.Device) { //nolint: interfacer
	if d.Type == device.GPU {
		gpuUUIDToContainerID.
			WithLabelValues(d.UUID, cID.String()).
			Inc()
	}
}

// DisassociateContainerGPU removes association between container ID and device ID.
func DisassociateContainerGPU(cID string, d device.Device) {
	if d.Type != device.GPU {
		return
	}

	gpuUUIDToContainerID.WithLabelValues(d.UUID, cID).Dec()
	gpuUUIDToContainerID.DeleteLabelValues(d.UUID, cID)
}

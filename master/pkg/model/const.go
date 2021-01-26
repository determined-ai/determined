package model

const (
	// DeterminedK8ContainerName is the name of the container that executes the task within Kubernetes
	// pods that are launched by Determined.
	DeterminedK8ContainerName = "determined-container"
	// DeterminedK8FluentContainerName is the name of the container running Fluent Bit in each pod.
	DeterminedK8FluentContainerName = "determined-fluent-container"

	// MinUserSchedulingPriority is the smallest priority users may specify.
	MinUserSchedulingPriority = 1
	// MaxUserSchedulingPriority is the largest priority users may specify.
	MaxUserSchedulingPriority = 99

	// BestCheckpointPolicy will checkpoint trials after validation if the validation is the best
	// validation so far.
	BestCheckpointPolicy = "best"
	// AllCheckpointPolicy will always checkpoint trials after validation.
	AllCheckpointPolicy = "all"
	// NoneCheckpointPolicy will not checkpoint trials after validations.
	NoneCheckpointPolicy = "none"
)

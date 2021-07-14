package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage = "determinedai/environments:py-3.7-pytorch-1.9-lightning-1.3-tf-2.4-cpu-2409e48"
	GPUImage = "determinedai/environments:cuda-11.1-pytorch-1.9-lightning-1.3-tf-2.4-gpu-2409e48"
)

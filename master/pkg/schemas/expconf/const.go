package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	DefaultCPUImage = "determinedai/environments:py-3.7-pytorch-1.7-tf-1.15-cpu-1c26118"
	DefaultGPUImage = "determinedai/environments:cuda-10.2-pytorch-1.7-tf-1.15-gpu-1c26118"
)

package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	DefaultCPUImage = "determinedai/environments:py-3.7-pytorch-1.7-lightning-1.2-tf-2.4-cpu-606fd02"
	DefaultGPUImage = "determinedai/environments:cuda-11.0-pytorch-1.7-lightning-1.2-tf-2.4-gpu-606fd02"
)

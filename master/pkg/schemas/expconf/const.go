package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	DefaultCPUImage = "determinedai/environments:py-3.6.9-pytorch-1.4-tf-1.15-cpu-067db2b"
	DefaultGPUImage = "determinedai/environments:cuda-10.0-pytorch-1.4-tf-1.15-gpu-067db2b"
)

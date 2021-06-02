package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	DefaultCPUImage = "andazhou/environments:py-3.7-pytorch-1.8.1-lightning-1.2-tf-2.4-cpu-254d511"
	DefaultGPUImage = "andazhou/environments:cuda-11.1-pytorch-1.8.1-lightning-1.2-tf.2.4-gpu-254d511"
)

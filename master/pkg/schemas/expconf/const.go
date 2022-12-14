package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage  = "determinedai/environments-dev:py-3.8-pytorch-1.10-tf-2.8-cpu-15f6132"
	CUDAImage = "determinedai/environments-dev:cuda-11.3-pytorch-1.10-tf-2.8-gpu-15f6132"
	ROCMImage = "determinedai/environments-dev:rocm-5.0-pytorch-1.10-tf-2.7-rocm-15f6132"
)

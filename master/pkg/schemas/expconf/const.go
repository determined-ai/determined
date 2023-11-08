package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage  = "determinedai/environments-dev:py-3.8-pytorch-1.12-tf-2.11-cpu-ae2bddf"
	CUDAImage = "determinedai/environments-dev:cuda-11.3-pytorch-1.12-tf-2.11-gpu-ae2bddf"
	ROCMImage = "determinedai/environments-dev:rocm-5.0-pytorch-1.10-tf-2.7-rocm-ae2bddf"
)

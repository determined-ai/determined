package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage  = "determinedai/environments-dev:py-3.9-pytorch-1.12-tf-2.11-cpu-ef2ebad"
	CUDAImage = "determinedai/environments-dev:cuda-11.3-pytorch-1.12-tf-2.11-gpu-ef2ebad"
	ROCMImage = "determinedai/environments-dev:rocm-5.0-pytorch-1.10-tf-2.7-rocm-ef2ebad"
)

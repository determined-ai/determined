package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage  = "determinedai/environments:py-3.8-pytorch-1.12-tf-2.8-cpu-7aa5364"
	CUDAImage = "determinedai/environments:cuda-11.3-pytorch-1.12-tf-2.8-gpu-7aa5364"
	ROCMImage = "determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-7aa5364"
)

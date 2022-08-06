package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage  = "determinedai/environments:py-3.8-pytorch-1.10-tf-2.8-cpu-9119094"
	CUDAImage = "determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-9119094"
	ROCMImage = "determinedai/environments:rocm-4.2-pytorch-1.9-tf-2.5-rocm-9119094"
)

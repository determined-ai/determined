package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage  = "determinedai/environments:py-3.8-pytorch-1.10-lightning-1.5-tf-2.8-cpu-97e4c60"
	CUDAImage = "determinedai/environments:cuda-11.3-pytorch-1.10-lightning-1.5-tf-2.8-gpu-97e4c60"
	ROCMImage = "determinedai/environments:rocm-4.2-pytorch-1.9-tf-2.5-rocm-97e4c60"
)

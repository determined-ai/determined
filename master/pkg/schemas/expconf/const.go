package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage  = "determinedai/environments:py-3.8-pytorch-1.9-lightning-1.3-tf-2.4-cpu-9f0cb26"
	CUDAImage = "determinedai/environments:cuda-11.1-pytorch-1.9-lightning-1.3-tf-2.4-gpu-9f0cb26"
	ROCMImage = "determinedai/environments:rocm-4.2-pytorch-1.9-tf-2.5-rocm-9f0cb26"
)

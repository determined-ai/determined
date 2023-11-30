package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage  = "determinedai/environments:py-3.9-pytorch-1.12-tf-2.11-cpu-622d512"
	CUDAImage = "determinedai/environments:cuda-11.3-pytorch-1.12-tf-2.11-gpu-622d512"
	ROCMImage = "determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-622d512"
)

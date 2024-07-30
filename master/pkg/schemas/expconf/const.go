package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage  = "determinedai/pytorch-ngc:0.35.0"
	CUDAImage = "determinedai/pytorch-ngc:0.35.0"
	ROCMImage = "determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-622d512"
)

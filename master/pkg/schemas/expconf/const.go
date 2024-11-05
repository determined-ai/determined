package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage  = "determinedai/pytorch-ngc:0.38.0"
	CUDAImage = "determinedai/pytorch-ngc:0.38.0"
	ROCMImage = "determinedai/environments:rocm-5.6-pytorch-1.3-tf-2.10-rocm-mpich-0736b6d"
)

// Default log policies values.
const (
	CUDAOOM         = "CUDA OOM"
	CUDAOOMPattern  = ".*CUDA out of memory.*"
	ECCError        = "ECC Error"
	ECCErrorPattern = ".*uncorrectable ECC error encountered.*"
)

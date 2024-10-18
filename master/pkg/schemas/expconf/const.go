package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage  = "determinedai/pytorch-ngc-dev:0736b6d"
	CUDAImage = "determinedai/pytorch-ngc-dev:0736b6d"
	ROCMImage = "determinedai/environments:rocm-5.6-pytorch-1.3-tf-2.10-rocm-mpich-0736b6d"
)

// Default log policy values.
const (
	ECCErrorPattern = ".*uncorrectable ECC error encountered.*"
	ECCErrorSignal  = "ECC Error"
	CUDAOOMPattern  = ".*CUDA out of memory.*"
	CUDAOOMSignal   = "CUDA OOM"
)

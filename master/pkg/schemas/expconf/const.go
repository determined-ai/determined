package expconf

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

// Default task environment docker image names.
const (
	CPUImage = "determinedai/environments:py-3.8-pytorch-1.9-lightning-1.3-tf-2.4-cpu-825e8ee"
	GPUImage = "determinedai/environments:cuda-11.1-pytorch-1.9-lightning-1.3-tf-2.4-gpu-825e8ee"
	// TODO XXX: update this once we've got the mainline build.
	ROCMImage = "determinedilia/environments:rocm-4.2-pytorch-1.9-rocm-e9238da"
)

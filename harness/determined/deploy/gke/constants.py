class defaults:
    GPU_TYPE = "nvidia-tesla-t4"
    GPUS_PER_NODE = 4
    ZONE = "us-west1-b"
    MASTER_MACHINE_TYPE = "n1-standard-16"
    AGENT_MACHINE_TYPE = "n1-standard-32"
    MAX_GPU_NODES = 4
    MAX_CPU_NODES = 4
    K8S_NVIDIA_DAEMON = "https://raw.githubusercontent.com/GoogleCloudPlatform/container-engine-accelerators/master/nvidia-driver-installer/cos/daemonset-preloaded.yaml"  # noqa: E501

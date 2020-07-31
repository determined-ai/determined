class defaults:

    AGENT_INSTANCE_TYPE = "n1-standard-32"
    DB_PASSWORD = "postgres"
    ENVIRONMENT_IMAGE = "pedl-environments-aaa3750"
    GPU_NUM = 8
    GPU_TYPE = "nvidia-tesla-k80"
    MASTER_INSTANCE_TYPE = "n1-standard-2"
    MAX_IDLE_AGENT_PERIOD = "10m"
    MAX_AGENT_STARTING_PERIOD = "10m"
    OPERATION_TIMEOUT_PERIOD = "5m"
    MAX_DYNAMIC_AGENTS = 5
    STATIC_AGENTS = 0
    MIN_CPU_PLATFORM_MASTER = "Intel Skylake"
    MIN_CPU_PLATFORM_AGENT = "Intel Broadwell"
    NETWORK = "det-default"
    PORT = 8080
    REGION = "us-west1"

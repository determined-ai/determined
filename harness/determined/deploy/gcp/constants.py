class defaults:

    AUX_AGENT_INSTANCE_TYPE = "n1-standard-4"
    COMPUTE_AGENT_INSTANCE_TYPE = "n1-standard-32"
    DB_PASSWORD = "postgres"
    ENVIRONMENT_IMAGE = "det-environments-24586f0"
    GPU_NUM = 8
    GPU_TYPE = "nvidia-tesla-k80"
    MASTER_INSTANCE_TYPE = "n1-standard-2"
    MAX_AUX_CONTAINERS_PER_AGENT = 100
    MAX_IDLE_AGENT_PERIOD = "10m"
    MAX_AGENT_STARTING_PERIOD = "20m"
    OPERATION_TIMEOUT_PERIOD = "5m"
    MIN_DYNAMIC_AGENTS = 0
    MAX_DYNAMIC_AGENTS = 5
    STATIC_AGENTS = 0
    MIN_CPU_PLATFORM_MASTER = "Intel Skylake"
    MIN_CPU_PLATFORM_AGENT = "Intel Broadwell"
    NETWORK = "det-default"
    PORT = 8080
    REGION = "us-west1"
    SCHEDULER_TYPE = "fair_share"
    PREEMPTION_ENABLED = False

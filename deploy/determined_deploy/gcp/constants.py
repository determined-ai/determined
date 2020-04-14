class defaults:

    AGENT_INSTANCE_TYPE = "n1-standard-32"
    DB_PASSWORD = "postgres"
    ENVIRONMENT_IMAGE = "pedl-environments-5b275a7"
    GPU_NUM = 8
    GPU_TYPE = "nvidia-tesla-k80"
    HASURA_SECRET = "secret"
    MASTER_INSTANCE_TYPE = "n1-standard-2"
    MAX_IDLE_AGENT_PERIOD = "10m"
    MAX_INSTANCES = 5
    MIN_CPU_PLATFORM_MASTER = "Intel Skylake"
    MIN_CPU_PLATFORM_AGENT = "Intel Broadwell"
    NETWORK = "det-default"
    PORT = 8080
    REGION = "us-west1"

class deployment_types:
    SIMPLE = "simple"
    SIMPLE_RDS = "simple-rds"
    SECURE = "secure"
    EFS = "efs"
    GENAI = "genai"
    FSX = "fsx"
    GOVCLOUD = "govcloud"
    DEPLOYMENT_TYPES = [SIMPLE, SECURE, EFS, FSX, GOVCLOUD, SIMPLE_RDS, GENAI]
    TYPE_TAG_KEY = "deployment-type"


class defaults:
    DEPLOYMENT_TYPE = deployment_types.SIMPLE
    DB_PASSWORD = "postgres"
    REGION = "us-west-2"
    STACK_TAG_KEY = "managed-by"
    STACK_TAG_VALUE = "determined"
    DB_SIZE = 20
    DB_INSTANCE_TYPE = "db.m7g.large"
    DB_SNAPSHOT = ""


class cloudformation:
    CLUSTER_ID = "ClusterId"
    EXTRA_TAGS = "ExtraTags"
    KEYPAIR = "Keypair"
    MASTER_ID = "MasterId"
    BASTION_ID = "BastionId"
    CHECKPOINT_BUCKET = "CheckpointBucket"
    MASTER_INSTANCE_TYPE = "MasterInstanceType"
    AUX_AGENT_INSTANCE_TYPE = "AuxAgentInstanceType"
    COMPUTE_AGENT_INSTANCE_TYPE = "ComputeAgentInstanceType"
    PUBLIC_IP_ADDRESS = "PublicIpAddress"
    PRIVATE_IP_ADDRESS = "PrivateIpAddress"
    SUBNET_ID_KEY = "SubnetId"
    INBOUND_CIDR = "InboundCIDRRange"
    DET_ADDRESS = "DeterminedAddress"
    MASTER_PORT = "MasterPort"
    MASTER_SCHEME = "MasterScheme"
    VERSION = "Version"
    DB_PASSWORD = "DBPassword"
    DB_INSTANCE_TYPE = "DBInstanceType"
    DB_SNAPSHOT = "DBSnapshot"
    DB_SIZE = "DBSize"
    ENABLE_CORS = "EnableCORS"
    MASTER_TLS_CERT = "MasterTLSCert"
    MASTER_TLS_KEY = "MasterTLSKey"
    MASTER_CERT_NAME = "MasterCertName"
    BOTO3_SESSION = "Boto3Session"
    AGENT_TAG_NAME = "AgentTagName"
    MAX_IDLE_AGENT_PERIOD = "MaxIdleAgentPeriod"
    MAX_AGENT_STARTING_PERIOD = "MaxAgentStartingPeriod"
    MAX_AUX_CONTAINERS_PER_AGENT = "MaxAuxContainersPerAgent"
    MIN_DYNAMIC_AGENTS = "MinDynamicAgents"
    MAX_DYNAMIC_AGENTS = "MaxDynamicAgents"
    LOG_GROUP = "LogGroup"
    REGION = "Region"
    SPOT_ENABLED = "SpotEnabled"
    SPOT_MAX_PRICE = "SpotMaxPrice"
    SCHEDULER_TYPE = "SchedulerType"
    PREEMPTION_ENABLED = "PreemptionEnabled"
    CPU_ENV_IMAGE = "CpuEnvImage"
    GPU_ENV_IMAGE = "GpuEnvImage"
    LOG_GROUP_PREFIX = "LogGroupPrefix"
    RETAIN_LOG_GROUP = "RetainLogGroup"
    IMAGE_REPO_PREFIX = "ImageRepoPrefix"
    MASTER_CONFIG_TEMPLATE = "MasterConfigTemplate"
    MOUNT_EFS_ID = "MountEFSId"
    MOUNT_FSX_ID = "MountFSxId"
    AGENT_REATTACH_ENABLED = "AgentReattachEnabled"
    AGENT_RECONNECT_ATTEMPTS = "AgentReconnectAttempts"
    AGENT_RECONNECT_BACKOFF = "AgentReconnectBackoff"
    AGENT_CONFIG_FILE_CONTENTS = "AgentConfigFileContents"
    MASTER_IMAGE_NAME = "MasterImageName"
    AGENT_IMAGE_NAME = "AgentImageName"
    DOCKER_USER = "DockerUser"
    DOCKER_PASS = "DockerPass"
    NOTEBOOK_TIMEOUT = "NotebookTimeout"
    GENAI_VERSION = "GenAIVersion"
    GENAI_SHARED_FS_MOUNT_PATH = "GenAISharedFSMountPath"
    INITIAL_USER_PASSWORD = "InitialUserPassword"


class misc:
    TEMPLATE_PATH = "determined.deploy.aws.templates"
    CLOUDFORMATION_REGEX = r"^[a-zA-Z][-a-zA-Z0-9]*$"
    SUPPORTED_REGIONS = [
        "ap-northeast-1",
        # TODO(DET-4258) Uncomment these when we fully support all P3 regions.
        # "ap-northeast-2",
        # "ap-southeast-1",
        # "ap-southeast-2",
        "eu-central-1",
        "eu-west-1",
        # "eu-west-2",
        "us-east-1",
        "us-east-2",
        "us-west-2",
        "us-gov-east-1",
        "us-gov-west-1",
    ]

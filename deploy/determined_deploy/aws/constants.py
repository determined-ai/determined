class deployment_types:
    SIMPLE = "simple"
    SECURE = "secure"
    VPC = "vpc"
    EFS = "efs"
    FSX = "fsx"
    DEPLOYMENT_TYPES = [SIMPLE, SECURE, VPC, EFS, FSX]


class defaults:
    DEPLOYMENT_TYPE = deployment_types.SIMPLE
    DB_PASSWORD = "postgres"
    REGION = "us-west-2"
    STACK_TAG_KEY = "managed-by"
    STACK_TAG_VALUE = "determined"


class cloudformation:
    CLUSTER_ID = "ClusterId"
    KEYPAIR = "Keypair"
    VPC = "VPC"
    PUBLIC_SUBNET = "PublicSubnetId"
    PRIVATE_SUBNET = "PrivateSubnetId"
    AGENT_INSTANCE_PROFILE_KEY = "AgentInstanceProfile"
    AGENT_SECURITY_GROUP_ID_KEY = "AgentSecurityGroupId"
    MASTER_ID = "MasterId"
    BASTION_ID = "BastionId"
    CHECKPOINT_BUCKET = "CheckpointBucket"
    MASTER_INSTANCE_TYPE = "MasterInstanceType"
    AGENT_INSTANCE_TYPE = "AgentInstanceType"
    PUBLIC_IP_ADDRESS = "PublicIpAddress"
    PRIVATE_IP_ADDRESS = "PrivateIpAddress"
    SUBNET_ID_KEY = "SubnetId"
    INBOUND_CIDR = "InboundCIDRRange"
    DET_ADDRESS = "DeterminedAddress"
    VERSION = "Version"
    DB_PASSWORD = "DBPassword"
    ENABLE_CORS = "EnableCORS"
    MASTER_TLS_CERT = "MasterTLSCert"
    MASTER_TLS_KEY = "MasterTLSKey"
    MASTER_CERT_NAME = "MasterCertName"
    BOTO3_SESSION = "Boto3Session"
    AGENT_TAG_NAME = "AgentTagName"
    MAX_IDLE_AGENT_PERIOD = "MaxIdleAgentPeriod"
    MAX_AGENT_STARTING_PERIOD = "MaxAgentStartingPeriod"
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


class misc:
    TEMPLATE_PATH = "determined_deploy.aws.templates"
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
    ]

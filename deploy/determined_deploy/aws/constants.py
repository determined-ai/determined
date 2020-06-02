class deployment_types:
    SIMPLE = "simple"
    SECURE = "secure"
    VPC = "vpc"
    DEPLOYMENT_TYPES = [SIMPLE, SECURE, VPC]


class defaults:
    DEPLOYMENT_TYPE = deployment_types.SIMPLE
    DB_PASSWORD = "postgres"
    REGION = "us-west-2"


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
    BOTO3_SESSION = "Boto3Session"
    AGENT_TAG_NAME = "AgentTagName"
    MAX_IDLE_AGENT_PERIOD = "MaxIdleAgentPeriod"
    MAX_AGENT_STARTING_PERIOD = "MaxAgentStartingPeriod"
    MAX_DYNAMIC_AGENTS = "MaxDynamicAgents"
    LOG_GROUP = "LogGroup"
    REGION = "Region"


class misc:
    TEMPLATE_PATH = "determined_deploy.aws.templates"
    CLOUDFORMATION_REGEX = r"^[a-zA-Z][-a-zA-Z0-9]*$"
    SUPPORTED_REGIONS = ["us-west-2", "us-east-1"]

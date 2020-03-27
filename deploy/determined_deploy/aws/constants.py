class deployment_types:
    SIMPLE = "simple"
    SECURE = "secure"
    VPC = "vpc"
    DEPLOYMENT_TYPES = [SIMPLE, SECURE, VPC]


class defaults:
    KEYPAIR_NAME = "determined-keypair"
    DEPLOYMENT_TYPE = deployment_types.SIMPLE
    DET_STACK_NAME_BASE = "determined-{}"
    DB_PASSWORD = "postgres"
    HASURA_SECRET = "hasura"


class cloudformation:
    USER_NAME = "UserName"
    KEYPAIR = "Keypair"
    VPC = "VPC"
    PUBLIC_SUBNET = "PublicSubnetId"
    PRIVATE_SUBNET = "PrivateSubnetId"
    BASTION_AMI = "BastionAmiId"
    MASTER_AMI = "MasterAmiId"
    AGENT_AMI = "AgentAmiId"
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
    HASURA_SECRET = "HasuraSecret"
    DET_STACK_NAME = "DeterminedStackName"
    BOTO3_SESSION = "Boto3Session"
    AGENT_TAG_NAME = "AgentTagName"
    MAX_IDLE_AGENT_PERIOD = "MaxIdleAgentPeriod"
    MAX_INSTANCES = "MaxInstances"


class misc:
    TEMPLATE_PATH = "determined_deploy.aws.templates"

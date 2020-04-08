class deployment_types:
    SIMPLE = "simple"
    SECURE = "secure"
    VPC = "vpc"
    DEPLOYMENT_TYPES = [SIMPLE, SECURE, VPC]


class defaults:
    KEYPAIR_NAME = "determined-keypair"
    MASTER_INSTANCE_TYPE = "t2.medium"
    AGENT_INSTANCE_TYPE = "p2.8xlarge"
    BASTION_AMI = "ami-06d51e91cea0dac8d"
    MASTER_AMI_ID = "ami-79873901"
    AGENT_AMI_ID = "ami-0c8bb82d0e2346768"
    DEPLOYMENT_TYPE = deployment_types.SIMPLE
    DET_STACK_NAME_BASE = "determined-{}"
    INBOUND_CIDR = "0.0.0.0/0"
    DB_PASSWORD = "postgres"
    HASURA_SECRET = "hasura"
    REGION = "us-west-2"
    MAX_IDLE_AGENT_PERIOD = "10m"
    MAX_INSTANCES = 5


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
    DET_MASTER_YAML_PATH = "/usr/local/determined/etc/master.yaml"
    TEMPLATE_PATH = "determined_deploy.aws.templates"

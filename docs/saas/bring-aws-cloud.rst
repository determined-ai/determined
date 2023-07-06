.. _deploy-aws-cloud:

###########################
 Bring Your Own Cloud: AWS
###########################

.. meta::
   :description: Steps for integrating your cloud provider account with Determined.

*****
 AWS
*****

Enabling Cross-Account Access
=============================

To grant Determined Cloud access to your AWS account, you will need to connect your AWS account with
the Determined Cloud. First, setup an `AWS credentials profile
<https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html>`__ for your AWS
account. Then, run the following detcloud command to create the role:

.. code::

   python -m detcloud.cli connect aws

The command above creates default roles and instance profiles. If you want to customize their
creation, run this command and replace the text delimited by ``<`` and ``>`` with desired values:

.. code::

   python -m detcloud.cli connect aws --xacct-role-name <cross-account role name> \
       --master-instance-profile-name <master instance profile name> \
       --agent-instance-profile-name <agent instance profile name>

To get an explaination of the command arguments, run:

.. code::

   python -m detcloud.cli connect aws --help

And it will print the command’s help as shown below (The output might change in newer versions of
the command without notice):

.. code::

   usage: __main__.py connect aws [-h] [-a ACCOUNT_ID] [--xacct-role-name XACCT_ROLE_NAME] [--master-instance-profile-name MASTER_INSTANCE_PROFILE_NAME]
                                  [--agent-instance-profile-name AGENT_INSTANCE_PROFILE_NAME]

   optional arguments:
     -h, --help            show this help message and exit
     -a ACCOUNT_ID, --account-id ACCOUNT_ID
                           ID of the Determined Account to connect with (default: 544296492693)
     --xacct-role-name XACCT_ROLE_NAME
                           Name of the cross-acount role to be assumed by Determined Cloud (default: det-cloud-customer-xacct-mgmt)
     --master-instance-profile-name MASTER_INSTANCE_PROFILE_NAME
                           Name of the master instance profile (default: det-cloud-customer-master)
     --agent-instance-profile-name AGENT_INSTANCE_PROFILE_NAME
                           Name of the agent instance profile (default: det-cloud-customer-agent)

If the command is run successfully, you will see output similar to the following:

.. code::

   Will connect your AWS account 123456789012 with Determined's AWS account 544296492693, proceed? (y/N)y
   IAM role created: arn:aws:iam::123456789012:role/det-cloud-customer-xacct-mgmt
   Instance profile created: arn:aws:iam::123456789012:instance-profile/det-cloud-customer-master
   Instance profile created: arn:aws:iam::123456789012:instance-profile/det-cloud-customer-agent

You will need to provide the info above when onboarding Determined Cloud.

**Note**: you will want to make note of the roles and instance profiles created so that you can
verify or reference them in the future.

How Determined Cloud Manages Your Clusters
==========================================

Once Determined Cloud has access to your account, it will be able to continuously manage Determined
clusters for you. Determined Cloud will create and manage these resources for a cluster:

-  Networking:

   -  Elastic IPs
   -  Gateways
   -  Network interfaces
   -  Route tables
   -  Route53 records
   -  Security groups
   -  Subnets
   -  VPCs

-  Compute:

   -  EC2 instances
   -  SSH key pairs

-  Storage:

   -  Aurora DB clusters
   -  S3 buckets

-  IAM:

   -  Roles
   -  Instance profiles
   -  Policies

-  Other:

   -  KMS keys
   -  CloudWatch log groups

In general, Determined Cloud performs these operations to the resources it manages:

-  Create
-  Modify
-  Delete
-  Connect multiple resources
-  Monitor resource status
-  Save logs where applicable
-  Create backups where applicable

In order for Determined Cloud to perform the aforementioned operations, your account *must* include
an IAM role that trusts Determined’s AWS account and includes the necessary permissions. It can be
set up by running a command as illustrated earlier, and the role’s details are listed in the next
section.

Determined Cloud is designed to manage both existing and new accounts. Existing resources in an
account usually do not affect the managed Determined clusters. However, in the situation below, you
should examine existing resources for potential issues:

-  Your existing resources use up a non-trivial portion of the account’s `quotas
   <https://docs.aws.amazon.com/general/latest/gr/aws_service_limits.html>`__, and that would reduce
   the amount of resources Determined Cloud can create. You can often increase the quota, but some
   resources quotas have limits on how much you can increase them.

-  Your existing resources interact with the Determined cluster. For example, you need to set up
   peering between the Determiend cluster’s VPC and an existing VPC. In this case, it is recommended
   that the two VPCs have non-overlapping IP ranges.

Required Role and Instance Profiles
===================================

If you choose to create the IAM Role and Policy manually, we will need the following permissions at
a minimum:

The Cross-Account Role
----------------------

Required Trust Relationship:

.. code::

   {
       "Version": "2012-10-17",
       "Statement": [
           {
               "Effect": "Allow",
               "Principal": {
                   "AWS": f"arn:aws:iam::544296492693:role/det-cloud-internal-global-mgmt-role"
               },
               "Action": "sts:AssumeRole",
           },
           {
               "Effect": "Allow",
               "Principal": {
                   "AWS": f"arn:aws:iam::544296492693:role/det-cloud-internal-aws-us-west-2-mgmt-role"
               },
               "Action": "sts:AssumeRole",
           }
       ],
   }

Note: these details are for the internal preview release at ``https://internal.det-cloud.net``. The
account number and names are subject to change and will vary between deployments.

Required Permissions Policy:

**Note**: you need to replaced the text delimited with ``<`` and ``>`` with desired values

.. code::

   {
       "Version": "2012-10-17",
       "Statement": [
           {
               "Sid": "DetCrossAccountAccess",
               "Effect": "Allow",
               "Action": [
                   "cloudwatch:GetMetricData",
                   "ec2:AllocateAddress",
                   "ec2:AssociateAddress",
                   "ec2:AssociateRouteTable",
                   "ec2:AttachInternetGateway",
                   "ec2:AuthorizeSecurityGroupEgress",
                   "ec2:AuthorizeSecurityGroupIngress",
                   "ec2:CreateInternetGateway",
                   "ec2:CreateNatGateway",
                   "ec2:CreateNetworkInterface",
                   "ec2:CreateRoute",
                   "ec2:CreateRouteTable",
                   "ec2:CreateSubnet",
                   "ec2:CreateTags",
                   "ec2:CreateVpc",
                   "ec2:DeleteInternetGateway",
                   "ec2:DeleteKeyPair",
                   "ec2:DeleteNatGateway",
                   "ec2:DeleteNetworkInterface",
                   "ec2:DeleteRouteTable",
                   "ec2:DeleteSubnet",
                   "ec2:DeleteVpc",
                   "ec2:DescribeAddresses",
                   "ec2:DescribeAvailabilityZones",
                   "ec2:DescribeInstanceStatus",
                   "ec2:DescribeInstanceTypes",
                   "ec2:DescribeInstances",
                   "ec2:DescribeInternetGateways",
                   "ec2:DescribeKeyPairs",
                   "ec2:DescribeNatGateways",
                   "ec2:DescribeNetworkInterfaces",
                   "ec2:DescribeRouteTables",
                   "ec2:DescribeSecurityGroups",
                   "ec2:DescribeSubnets",
                   "ec2:DescribeVpcs",
                   "ec2:DetachInternetGateway",
                   "ec2:DisassociateRouteTable",
                   "ec2:ImportKeyPair",
                   "ec2:ReleaseAddress",
                   "ec2:RunInstances",
                   "ec2:TerminateInstances",
                   "iam:AddRoleToInstanceProfile",
                   "iam:AttachRolePolicy",
                   "iam:CreateInstanceProfile",
                   "iam:CreatePolicy",
                   "iam:CreateRole",
                   "iam:DeleteInstanceProfile",
                   "iam:DeletePolicy",
                   "iam:DeleteRole",
                   "iam:DetachRolePolicy",
                   "iam:GetInstanceProfile",
                   "iam:ListPolicies",
                   "iam:ListRoles",
                   "iam:RemoveRoleFromInstanceProfile",
                   "iam:SimulatePrincipalPolicy",
                   "iam:TagInstanceProfile",
                   "iam:TagPolicy",
                   "iam:TagRole",
                   "kms:CreateGrant",
                   "kms:DescribeKey",
                   "logs:CreateLogGroup",
                   "logs:DeleteLogGroup",
                   "logs:PutRetentionPolicy",
                   "logs:TagResource",
                   "rds:AddTagsToResource",
                   "rds:CreateDBCluster",
                   "rds:CreateDBSubnetGroup",
                   "rds:DeleteDBCluster",
                   "rds:DeleteDBSubnetGroup",
                   "rds:DescribeDBClusters",
                   "rds:ModifyDBCluster",
                   "rds:RestoreDBClusterToPointInTime",
                   "s3:CreateBucket",
                   "s3:DeleteBucket",
                   "s3:PutBucketPolicy",
                   "s3:PutEncryptionConfiguration",
                   "servicequotas:GetServiceQuota",
                   "ssm:GetCommandInvocation",
                   "ssm:SendCommand",
                   "ssm:StartSession",
                   "ssm:DeleteParameter",
                   "ssm:PutParameter",
               ],
               "Resource": "*",
           },
           {
               "Sid": "DetCrossPassRole",
               "Effect": "Allow",
               "Action": "iam:PassRole",
               "Resource": "arn:aws:iam::<your AWS account ID>:role/<master instance profile name>",
           },
       ],
   }

The Master Instance Profile
---------------------------

Required Trust Relationship:

.. code::

   {
       "Version": "2012-10-17",
       "Statement": [
           {
               "Effect": "Allow",
               "Principal": {"Service": "ec2.amazonaws.com"},
               "Action": "sts:AssumeRole",
           }
       ],
   }

Required Permissions Policy:

**Note**: you need to replaced the text delimited with ``<`` and ``>`` with desired values

.. code::

   {
       "Version": "2012-10-17",
       "Statement": [
           {
               "Action": [
                   "ec2:DescribeInstances",
                   "ec2:TerminateInstances",
                   "ec2:CreateTags",
                   "ec2:RunInstances",
                   "ec2:CancelSpotInstanceRequests",
                   "ec2:RequestSpotInstances",
                   "ec2:DescribeSpotInstanceRequests",
                   "logs:CreateLogStream",
                   "logs:PutLogEvents",
               ],
               "Effect": "Allow",
               "Resource": "*",
           },
           {
               "Action": "iam:PassRole",
               "Resource": "arn:aws:iam::<your AWS account ID>:role/<agent instance profile name>",
               "Effect": "Allow",
           },
       ],
   }

Also include the AWS managed policy ``AmazonSSMManagedEC2InstanceDefaultPolicy``.

The Agent Instance Profile
--------------------------

Required Trust Relationship:

.. code::

   {
       "Version": "2012-10-17",
       "Statement": [
           {
               "Effect": "Allow",
               "Principal": {"Service": "ec2.amazonaws.com"},
               "Action": "sts:AssumeRole",
           }
       ],
   }

Required Permissions Policy:

.. code::

   {
       "Version": "2012-10-17",
       "Statement": [
           {
               "Action": [
                   "s3:*",
                   "ec2:DescribeInstances",
                   "logs:CreateLogStream",
                   "logs:PutLogEvents",
               ],
               "Effect": "Allow",
               "Resource": "*",
           }
       ],
   }

Also include the AWS managed policy ``AmazonSSMManagedEC2InstanceDefaultPolicy``.

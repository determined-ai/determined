.. _install-aws:

####################
 Install Determined
####################

This user guide describes how to deploy a Determined cluster on Amazon Web Services (AWS). The
:ref:`det deploy <determined-deploy>` tool makes it easy to create and install these resources. If
you would rather create the cluster manually, see the :ref:`aws-manual-deployment` section below.

For more information about using Determined on AWS, see the :ref:`topic_guide_aws` topic guide.

.. _determined-deploy:

*********************
 ``det deploy`` tool
*********************

The ``det deploy`` tool is provided by the ``determined`` Python package. It uses `AWS
CloudFormation <https://aws.amazon.com/cloudformation/>`__ to automatically deploy and configure a
Determined cluster. CloudFormation builds the necessary components for Determined into a single
CloudFormation stack.

Requirements
============

-  Either AWS credentials or an IAM role with permissions to access AWS CloudFormation APIs. See the
   `AWS Documentation <https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html>`__
   for information on how to use AWS credentials.

-  An `AWS EC2 Keypair <https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html>`__.

You may also want to increase the `EC2 instance limits
<https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-resource-limits.html>`__ on your account
--- the `default instance limits
<https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-on-demand-instances.html#ec2-on-demand-instances-limits>`__
on GPU instances are zero. The default configuration for ``det deploy`` can result in launching up
to 5 ``g4dn.metal`` instances (which have 94 vCPUs each), which would exceed the default quota. AWS
instance limits can be increased by submitting a request to the `AWS Support Center
<https://console.aws.amazon.com/support/home?#/case/create?issueType=service-limit-increase&limitType=service-code-ec2-instances>`__.

Installation
============

``det`` command line tool can be installed using ``pip``:

.. code::

   pip install determined

.. include:: ../../../_shared/note-pip-install-determined.txt

Deploying
=========

The basic command to deploy a cluster is as follows:

.. code::

   det deploy aws up --cluster-id CLUSTER_ID --keypair KEYPAIR_NAME

``CLUSTER_ID`` is an arbitrary unique ID for the new cluster. We recommend choosing a cluster ID
that is memorable and helps identify what the cluster is being used for. The cluster ID will be used
as the AWS CloudFormation stack name.

``KEYPAIR_NAME`` is the name of the AWS EC2 key pair to use when provisioning the cluster. If the
AWS CLI is installed on your machine, you can get a list of the available keypair names by running
``aws ec2 describe-key-pairs``.

The deployment process may take 5--10 minutes. When it completes, summary information about the
newly deployed cluster will be printed, including the URL of the Determined master.

.. _determined-deploy-deployment-types:

Deployment Types
================

``det deploy`` supports multiple deployment types to work with different security needs. The
deployment type can be specified using the ``--deployment-type`` argument (e.g., ``det deploy aws
--deployment-type secure``).

-  ``simple``: The simple deployment provides an easy way to deploy a Determined cluster in AWS.
   This creates the master instance in the default subnet for the account.

-  ``efs``: The EFS deployment creates an `EFS <https://aws.amazon.com/efs/>`_ file system and a
   Determined cluster into a separate VPC. The EFS drive attaches to agent instances at
   ``/mnt/efs``. This path is automatically bind-mounted into the task containers.

-  ``fsx``: The FSX deployment creates a `Lustre FSx <https://aws.amazon.com/fsx/lustre/>`_ file
   system and a Determined cluster into a separate VPC. The FSx drive attaches to agent instances at
   ``/mnt/fsx``. This path is automatically bind-mounted into the task containers.

-  ``secure``: The secure deployment creates resources to lock down the Determined cluster. These
   resources are:

   -  A VPC with a public and private subnet
   -  A NAT gateway for the private subnet to make outbound connections
   -  An S3 VPC gateway so the private subnet can access S3
   -  A bastion instance in the public subnet
   -  A master instance in the private subnet

CLI Arguments
=============

Spinning up or updating the Cluster
-----------------------------------

.. code::

   det deploy aws up --cluster-id CLUSTER_ID --keypair KEYPAIR_NAME

.. list-table::
   :widths: 25 50 25
   :header-rows: 1

   -  -  Argument
      -  Description
      -  Default Value

   -  -  ``--cluster-id``
      -  Unique ID for the cluster (used as the CloudFormation stack name).
      -  *required*

   -  -  ``--keypair``
      -  The name of the AWS EC2 key pair to use for both master and agent instances.
      -  *required*

   -  -  ``--region``
      -  AWS region to deploy into.
      -  The default region for the AWS user

   -  -  ``--profile``
      -  AWS profile to use for deploying cluster resources.
      -  default

   -  -  ``--master-instance-type``
      -  AWS instance type to use for the master.
      -  m5.large

   -  -  ``--aux-agent-instance-type``

      -  AWS instance type to use for the agents in the auxiliary resource pool. Must be one of the
         following instance types: ``g4dn.xlarge``, ``g4dn.2xlarge``, ``g4dn.4xlarge``,
         ``g4dn.8xlarge``, ``g4dn.16xlarge``, ``g4dn.12xlarge``, ``g4dn.metal``, ``g5.xlarge``,
         ``g5.2xlarge``, ``g5.4xlarge``, ``g5.8xlarge``, ``g5.12xlarge``, ``g5.16xlarge``,
         ``g5.24xlarge``, ``g5.48large``, ``p3.2xlarge``, ``p3.8xlarge``, ``p3.16xlarge``,
         ``p3dn.24xlarge``, ``p4d.24xlarge``, or any general purpose instance types (``t2``, ``t3``,
         ``c4``, ``c5``, ``m4``, ``m5`` and variants).

      -  t2.xlarge

   -  -  ``--compute-agent-instance-type``

      -  AWS instance type to use for the agents in the compute resource pool. For GPU-based
         training, must be one of the following instance types: ``g4dn.xlarge``, ``g4dn.2xlarge``,
         ``g4dn.4xlarge``, ``g4dn.8xlarge``, ``g4dn.16xlarge``, ``g4dn.12xlarge``, ``g4dn.metal``,
         ``g5.xlarge``, ``g5.2xlarge``, ``g5.4xlarge``, ``g5.8xlarge``, ``g5.12xlarge``,
         ``g5.16xlarge``, ``g5.24xlarge``, ``g5.48large``, ``p3.2xlarge``, ``p3.8xlarge``,
         ``p3.16xlarge``, ``p3dn.24xlarge``, or ``p4d.24xlarge``. For CPU-based training or testing,
         any general purpose instance type may be used (``t2``, ``t3``, ``c4``, ``c5``, ``m4``,
         ``m5`` and variants).

      -  g4dn.metal

   -  -  ``--deployment-type``
      -  The :ref:`deployment type <determined-deploy-deployment-types>` to use.
      -  simple

   -  -  ``--inbound-cidr``
      -  CIDR range for inbound traffic.
      -  0.0.0.0/0

   -  -  ``--db-password``
      -  The password for ``postgres`` user for database.
      -  postgres

   -  -  ``--max-aux-containers-per-agent``
      -  The maximum number of containers to launch on each agent in the default auxiliary
         :ref:`resource pool <resource-pools>`.
      -  100

   -  -  ``--max-idle-agent-period``
      -  The length of time to wait before idle dynamic agents will be automatically terminated.
      -  10m (10 minutes)

   -  -  ``--max-dynamic-agents``
      -  Maximum number of dynamic agent instances in the default compute :ref:`resource pool
         <resource-pools>`.
      -  5

   -  -  ``--spot``
      -  Use spot instances for the default auxiliary and compute resource pools.
      -  False

   -  -  ``--spot-max-price``

      -  The maximum price to use when launching spot instances. If the current spot market price
         exceeds this value, Determined will not create new instances. If no maximum price is
         configured, the maximum price will be the on-demand price for the configured instance type
         and region.

      -  Not set

   -  -  ``--dry-run``
      -  Print the template but do not execute it.
      -  False

   -  -  ``--master-config-template-path``
      -  Path to the custom ``master.yaml`` template. Default template can be obtained using ``det
         deploy aws dump-master-config-template``.
      -  Not set

   -  -  ``--efs-id``

      -  Preexisting EFS file system that will be mounted into the task containers; if not provided,
         a new EFS instance will be created. The agents must be able to connect to the EFS instance.
         This option can only be used together with the ``efs`` :ref:`deployment type
         <determined-deploy-deployment-types>`.

      -  Not set

   -  -  ``--fsx-id``

      -  Preexisting FSx file system that will be mounted into the task containers; if not provided,
         a new FSx instance will be created. The agents must be able to connect to the FSx instance.
         This option can only be used together with the ``fsx`` :ref:`deployment type
         <determined-deploy-deployment-types>`.

      -  Not set

   -  -  ``--shut-down-on-connection-loss``, ``--no-shut-down-on-connection-loss``
      -  Whether or not agent instances should automatically shut down when they lose connection to
         the master.
      -  Shut down automatically

Tearing Down the Cluster
------------------------

.. code::

   det deploy aws down --cluster-id CLUSTER_ID

.. list-table::
   :widths: 25 50 25
   :header-rows: 1

   -  -  Argument
      -  Description
      -  Default Value

   -  -  ``--cluster-id``
      -  Unique ID for the cluster (used as the CloudFormation stack name).
      -  *required*

   -  -  ``--region``
      -  AWS region deployed into.
      -  The default region for the AWS user

   -  -  ``--profile``
      -  AWS profile used for deploying cluster resources.
      -  default

Listing Clusters
----------------

.. code::

   det deploy aws list

.. list-table::
   :widths: 25 50 25
   :header-rows: 1

   -  -  Argument
      -  Description
      -  Default Value

   -  -  ``--region``
      -  AWS region to deploy into.
      -  The default region for the AWS user

   -  -  ``--profile``
      -  AWS profile used for deploying cluster resources.
      -  default

Printing the default ``master.yaml`` template
---------------------------------------------

.. code::

   det deploy aws dump-master-config-template

.. _aws-master-yaml-template:

Custom master.yaml templates
============================

Advanced users who require a deep customization of master settings (i.e., the ``master.yaml`` config
file) can use the ``master.yaml`` templating feature. Since ``det deploy aws`` fills in plenty of
infrastructure-related values such as VPC subnet ids or IAM instance profile roles, we provide a
simplified templating solution, similar to :ref:`helm charts in kubernetes <install-on-kubernetes>`.
Template language is based on golang templates, and includes ``sprig`` helper library and ``toYaml``
serialization helper.

Example workflow:

#. Get the default template using

   .. code::

      det deploy aws dump-master-config-template > /path/to/master.yaml.tmpl

#. Customize the template as you see fit by editing it in any text editor. For example, let's say a
   user wants to utilize (default) ``g4dn.metal`` 8-GPU instances for the :ref:`default compute pool
   <resource-pools>`, but they also often run single-GPU notebook jobs, for which a single
   ``g4dn.xlarge`` instance would be perfect. So, you want to add a third pool ``compute-pool-solo``
   with a customized instance type.

   Start with the default template, and find the ``resource_pools`` section:

   .. code:: yaml

      resource_pools:
        - pool_name: aux-pool
          max_aux_containers_per_agent: {{ .resource_pools.pools.aux_pool.max_aux_containers_per_agent }}
          provider:
            instance_type: {{ .resource_pools.pools.aux_pool.instance_type }}
            {{- toYaml .resource_pools.aws | nindent 6}}

        - pool_name: compute-pool
          max_aux_containers_per_agent: 0
          provider:
            instance_type: {{ .resource_pools.pools.compute_pool.instance_type }}
            cpu_slots_allowed: true
            {{- toYaml .resource_pools.aws | nindent 6}}

   Then, append a new section:

   .. code:: yaml

      - pool_name: compute-pool-solo
        max_aux_containers_per_agent: 0
        provider:
          instance_type: g4dn.xlarge
          {{- toYaml .resource_pools.aws | nindent 6}}

#. Use the new template:

   .. code::

      det deploy aws <ALL PREVIOUSLY USED FLAGS> --master-config-template-path /path/to/edited/master.yaml.tmpl

#. All set! Check the `Cluster` page in WebUI to ensure your cluster has 3 resource pools. In case
   of errors, ssh to the master instance as instructed by ``det deploy aws`` output, and check
   ``/var/log/cloud-init-output.log`` or ``sudo docker logs determined-master``.

.. _aws-modifying-deployment:

Modifying a Deployment
======================

To modify an already deployed cluster you have a few options:

#. If what you'd like to change is provided as a ``det deploy`` CLI option, you can re-deploy the
   cluster using ``det deploy``. Use the same full ``det deploy`` command as on cluster creation,
   but update the options as necessary, while keeping the ``cluster-id`` the same. ``det deploy``
   will then find the existing cluster, take it down, and spin up a new one with the updated
   options.

#. If you want more control over the master configuration while minimizing downtime, you can SSH
   into the master instance using the private key from the keypair that was used to provision the
   cluster. Once you're successfully connected, you can modify ``master.yaml`` under
   ``/usr/local/determined/etc`` and restart the master Docker container for your changes to take
   effect:

   .. code:: bash

      sudo docker container restart determined-master

   For example, if you want to add or modify resource pools you can edit the master configuration
   file at ``/usr/local/determined/etc/master.yaml`` and add a new resource pool entry.

.. _aws-manual-deployment:

*******************
 Manual Deployment
*******************

Database
========

Determined requires a PostgreSQL-compatible database, such as AWS Aurora. Configure the cluster to
use the database by including the database information in ``master.yaml``. Make sure to create a
database before running the Determined cluster (e.g., ``CREATE DATABASE <database-name>``).

Example ``master.yaml`` snippet:

.. code:: yaml

   db:
     user: "${database-user}"
     password: "${database-password}"
     host: "${database-hostname}"
     port: 5432
     name: "${database-name}"

Security Groups
===============

VPC Security Groups provide a set of rules for inbound and outbound network traffic. The
requirements for a Determined cluster are:

Master
------

-  Egress on all ports to agent security group
-  Egress on all ports to the Internet
-  Ingress on port 8080 for access the Determined WebUI and REST APIs
-  Ingress on port 22 for SSH (recommended but not required)
-  Ingress on all ports from agent security group

Example:

.. code:: yaml

   MasterSecurityGroupEgress:
     Type: AWS::EC2::SecurityGroupEgress
     Properties:
       GroupId: !GetAtt MasterSecurityGroup.GroupId
       DestinationSecurityGroupId: !GetAtt AgentSecurityGroup.GroupId
       FromPort: 0
       ToPort: 65535
       IpProtocol: tcp

   MasterSecurityGroupInternet:
     Type: AWS::EC2::SecurityGroupEgress
     Properties:
       GroupId: !GetAtt MasterSecurityGroup.GroupId
       CidrIp: 0.0.0.0/0
       FromPort: 0
       ToPort: 65535
       IpProtocol: tcp

   MasterSecurityGroupIngress:
     Type: AWS::EC2::SecurityGroupIngress
     Properties:
       GroupId: !GetAtt MasterSecurityGroup.GroupId
       FromPort: 8080
       ToPort: 8080
       IpProtocol: tcp
       SourceSecurityGroupId: !GetAtt AgentSecurityGroup.GroupId

   MasterSecurityGroupIngressUI:
     Type: AWS::EC2::SecurityGroupIngress
     Properties:
       GroupId: !GetAtt MasterSecurityGroup.GroupId
       FromPort: 8080
       ToPort: 8080
       IpProtocol: tcp
       CidrIp: !Ref InboundCIDRRange

   MasterSSHIngress:
     Type: AWS::EC2::SecurityGroupIngress
     Properties:
       GroupId: !GetAtt MasterSecurityGroup.GroupId
       IpProtocol: tcp
       FromPort: 22
       ToPort: 22
       CidrIp: !Ref InboundCIDRRange

Agent
-----

-  Egress on all ports to the Internet
-  Ingress on all ports from master security group
-  Ingress on all ports from agent security group
-  Ingress on port 22 for SSH (recommended but not required)

Example:

.. code:: yaml

   AgentSecurityGroupEgress:
     Type: AWS::EC2::SecurityGroupEgress
     Properties:
       GroupId: !GetAtt AgentSecurityGroup.GroupId
       CidrIp: 0.0.0.0/0
       FromPort: 0
       ToPort: 65535
       IpProtocol: tcp

   AgentSecurityGroupIngressMaster:
     Type: AWS::EC2::SecurityGroupIngress
     Properties:
       GroupId: !GetAtt AgentSecurityGroup.GroupId
       FromPort: 0
       ToPort: 65535
       IpProtocol: tcp
       SourceSecurityGroupId: !GetAtt MasterSecurityGroup.GroupId

   AgentSecurityGroupIngressAgent:
     Type: AWS::EC2::SecurityGroupIngress
     Properties:
       GroupId: !GetAtt AgentSecurityGroup.GroupId
       FromPort: 0
       ToPort: 65535
       IpProtocol: tcp
       SourceSecurityGroupId: !GetAtt AgentSecurityGroup.GroupId

   AgentSSHIngress:
     Type: AWS::EC2::SecurityGroupIngress
     Properties:
       GroupId: !GetAtt AgentSecurityGroup.GroupId
       IpProtocol: tcp
       FromPort: 22
       ToPort: 22
       CidrIp: !Ref InboundCIDRRange

IAM Roles
=========

IAM roles comprise IAM policies, which provide access to AWS APIs such as the EC2 or S3 API. The IAM
policies needed for the Determined cluster are:

Master
------

-  Allow EC2 to assume role
-  Allow EC2 to describe, create, and terminate instances with agent role
-  Allow EC2 to describe, create, and terminate spot instance requests (only required if using spot
   instances)

.. code:: yaml

   MasterRole:
     Type: AWS::IAM::Role
     Properties:
       AssumeRolePolicyDocument:
         Version: 2012-10-17
         Statement:
           - Effect: Allow
             Principal:
               Service:
                 - ec2.amazonaws.com
             Action:
               - sts:AssumeRole
       Policies:
         - PolicyName: determined-agent-policy
           PolicyDocument:
             Version: 2012-10-17
             Statement:
               - Effect: Allow
                 Action:
                   - ec2:DescribeInstances
                   - ec2:TerminateInstances
                   - ec2:CreateTags
                   - ec2:RunInstances
                   - ec2:CancelSpotInstanceRequests      # Only required if using spot instances
                   - ec2:RequestSpotInstances            # Only required if using spot instances
                   - ec2:DescribeSpotInstanceRequests    # Only required if using spot instances
                 Resource: "*"
         - PolicyName: pass-role
           PolicyDocument:
             Version: 2012-10-17
             Statement:
               - Effect: Allow
                 Action: iam:PassRole
                 Resource: !GetAtt AgentRole.Arn

Agent
-----

-  Allow EC2 to assume role
-  Allow S3 access for checkpoint storage
-  Allow agent instance to describe instances

.. code:: yaml

   AgentRole:
     Type: AWS::IAM::Role
     Properties:
       AssumeRolePolicyDocument:
         Version: 2012-10-17
         Statement:
           - Effect: Allow
             Principal:
               Service:
                 - ec2.amazonaws.com
             Action:
               - sts:AssumeRole
       Policies:
         - PolicyName: agent-s3-policy
           PolicyDocument:
             Version: 2012-10-17
             Statement:
               - Effect: Allow
                 Action: "s3:*"
                 Resource: "*"
         - PolicyName: determined-ec2
           PolicyDocument:
             Version: 2012-10-17
             Statement:
               - Effect: Allow
                 Action:
                   - ec2:DescribeInstances
                 Resource: "*"

Master Node
===========

The master node should be deployed on an EC2 instance with at least 4 CPUs (Intel Broadwell or
later), 8GB of RAM, and 200GB of disk storage. This roughly corresponds to an EC2 t2.large instance
or better. The AMI should be the default Ubuntu 18.04 AMI.

To run Determined:

#. Install Docker and create the ``determined`` Docker network.

   .. code::

      apt-get remove docker docker-engine docker.io containerd runc
      apt-get update
      apt-get install -y \
        apt-transport-https \
        ca-certificates \
        curl \
        gnupg-agent \
        software-properties-common
      curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
      add-apt-repository \
        "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
        $(lsb_release -cs) \
        stable"
      apt-get update
      apt-get install -y docker-ce docker-ce-cli containerd.io

      docker network create determined

#. Configure the cluster with ``master.yaml``. See :ref:`cluster-configuration` for more
   information.

   Notes:

   -  ``image_id`` should be the latest Determined agent AMI.
   -  ``instance_type`` should be any p2 or p3 EC2 instance type.
   -  For more information about resource pools, see :ref:`resource-pools`

   .. warning::

      An important assumption of Determined with dynamic agents is that any EC2 instances with the
      configured tag_key:tag_value pair are managed by the Determined master. This pair should be
      unique to your Determined installation. If it is not, Determined may inadvertently manage your
      non-Determined EC2 instances.

      If using spot instances, Determined also assumes that any EC2 spot instance requests with the
      configured tag_key:tag_value pair are managed by the Determined master.

   .. code:: yaml

      checkpoint_storage:
        type: s3
        bucket: ${CheckpointBucket}

      db:
        user: postgres
        password: "${DBPassword}"
        host: "${Database.Endpoint.Address}"
        port: 5432
        name: determined

      resource_pools:
        - pool_name: default
          description: The default resource pool
          provider:
            iam_instance_profile_arn: ${AgentInstanceProfile.Arn}
            image_id: ${AgentAmiId}
            agent_docker_image: determinedai/determined-agent:${Version}
            instance_name: determined-agent-${UserName}
            instance_type: ${AgentInstanceType}
            master_url: http://local-ipv4:8080
            max_idle_agent_period: ${MaxIdleAgentPeriod}
            max_instances: ${MaxInstances}
            network_interface:
              public_ip: true
              security_group_id: ${AgentSecurityGroup.GroupId}
            type: aws
            ssh_key_name: ${Keypair}
            tag_key: determined-${UserName}
            tag_value: determined-${UserName}-agent

#. Start the Determined master.

   .. code::

      docker run \
        --rm \
        --network determined \
        -p 8080:8080 \
        -v master.yaml:/etc/determined/master.yaml \
        determinedai/determined-master:${Version}

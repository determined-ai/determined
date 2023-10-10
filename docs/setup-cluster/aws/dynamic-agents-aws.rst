.. _dynamic-agents-aws:

#######################################
 Deploy Determined with Dynamic Agents
#######################################

This document describes how to install, configure, and upgrade a deployment of Determined with
dynamic agents on AWS. See :ref:`elastic-infrastructure` for an overview of using elastic
infrastructure in Determined.

Determined is able to launch dynamic agents as spot instances, which can be much less costly than
using standard on-demand instances. For more details on spot instances, see :ref:`aws-spot`.

*********************
 System Requirements
*********************

EC2 Instance Tags
=================

An important assumption of Determined with dynamic agents is that any EC2 instances with the
configured ``tag_key:tag_value`` pair are managed by the Determined master (See
:ref:`aws-cluster-configuration`). This pair should be unique to your Determined installation. If it
is not, Determined may inadvertently manage your non-Determined EC2 instances.

If using spot instances, Determined also assumes that any EC2 spot instance requests with the
configured ``tag_key:tag_value`` pair are managed by the Determined master.

EC2 AMIs
========

-  The Determined master node will run on a custom AMI that will be shared with you by Determined
   AI.
-  Determined agent nodes will run on a custom AMI that will be shared with you by Determined AI.

EC2 Instance Types
==================

-  The Determined master node should be deployed on an EC2 instance supporting >= 2 CPUs (Intel
   Broadwell or later), 4GB of RAM, and 100GB of disk storage. This corresponds to an EC2
   ``t2.medium`` instance or better.

-  All Determined agent nodes must be the same AWS instance type; any G4, P2, or P3 instance type is
   supported. This can be configured in the :ref:`aws-cluster-configuration`.

.. _master-iam-role:

Master IAM Role
===============

The Determined master needs to have an IAM role with the following permissions:

-  ``ec2:CreateTags``: used to tag the Determined agent instances that the Determined master
   provisions. These tags are configured by the `aws-cluster-configuration`.
-  ``ec2:DescribeInstances``: used to find active Determined agent instances based on tags.
-  ``ec2:RunInstances``: used to provision Determined agent instances.
-  ``ec2:TerminateInstances``: used to terminate idle Determined agent instances.

If using spot instances, the master also needs the following permissions:

-  ``ec2:RequestSpotInstances``: used to provision Determined agent instances as spot instances.
-  ``ec2:CancelSpotInstanceRequests``: used to adjust the number of spot instance requests to match
   the number of instances needed for the current workloads.
-  ``ec2:DescribeSpotInstanceRequests``: used to find open spot instance requests that, once
   fulfilled, will create Determined agent spot instances.

An example IAM policy with the appropriate permissions is below:

.. code:: json

   {
     "Version": "2012-10-17",
     "Statement": [
        {
          "Sid": "VisualEditor0",
          "Effect": "Allow",
          "Action": [
            "ec2:DescribeInstances",
            "ec2:TerminateInstances",
            "ec2:CreateTags",
            "ec2:RunInstances",
            "ec2:CancelSpotInstanceRequests",
            "ec2:RequestSpotInstances",
            "ec2:DescribeSpotInstanceRequests",
          ],
          "Resource": "*"
        }
     ]
   }

If you need to attach an instance profile to the agent (e.g., ``iam_instance_profile_arn`` is set in
the :ref:`aws-cluster-configuration`), make sure to add ``PassRole`` policy to the master role with
``Resource`` set to the desired agent role. For example:

.. code:: json

   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": "iam:PassRole",
         "Resource": "<arn::agent-role>"
       }
     ]
   }

See `Using an IAM Role to Grant Permissions to Applications Running on Amazon EC2 Instances
<https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2.html>`__ for details.

.. _aws-network-requirements:

Set up Internet Access
======================

-  The Determined Docker images are hosted on Docker Hub. Determined agents need access to Docker
   Hub for such tasks as building new images for user workloads.

-  If packages, data, or other resources needed by user workloads are hosted on the public Internet,
   Determined agents need to be able to access them. Note that agents can be :ref:`configured to use
   proxies <agent-network-proxy>` when accessing network resources.

-  For best performance, it is recommended that the Determined master and agents use the same
   physical network or VPC. When using VPCs on a public cloud provider, you may need to take
   additional steps to ensure instances in the VPC can access the Internet:

   -  On GCP, either the instances must have an external IP address or a `GCP Cloud NAT
      <https://cloud.google.com/nat/docs/overview>`_ should be configured for the VPC.

   -  On AWS, the instances must have a public IP address and a `VPC Internet Gateway
      <https://docs.aws.amazon.com/vpc/latest/userguide/VPC_Internet_Gateway.html>`_ should be
      configured for the VPC.

Set up Firewall Rules
=====================

The firewall rules must satisfy the following network access requirements for the master and agents.

Master
------

-  Inbound TCP to the master's network port from the Determined agent instances, as well as all
   machines where developers want to use the Determined CLI or WebUI. The default port is ``8443``
   if TLS is enabled and ``8080`` if not.

-  Outbound TCP to all ports on the Determined agents.

Agents
------

-  Inbound TCP from all ports on the master to all ports on the agent.

-  Outbound TCP from all ports on the agent to the master's network port.

-  Outbound TCP to the services that host the Docker images, packages, data, and other resources
   that need to be accessed by user workloads.

   -  For example, if your data is stored on Amazon S3, ensure the firewall rules allow access to
      this data.

-  Inbound and outbound TCP on all ports to and from each Determined agent. The details are as
   follows:

   -  Inbound and outbound TCP ports 1734 and 1750 are used for synchronization between trial
      containers.

   -  Inbound and outbound TCP port 12350 is used for internal SSH-based communication between trial
      containers.

   -  Inbound and outbound TCP port 12355 is used for GLOO rendezvous between trial containers.

   -  Inbound and outbound ephemeral TCP ports in the range 1024-65536 are used for communication
      between trials via GLOO. This range is configured by the configuration parameter
      ``task_container_defaults.gloo_port_range`` inside ``master.yaml`` as described in the
      :ref:`cluster-configuration` guide.

   -  For every GPU on each agent machine, an inbound and outbound ephemeral TCP port in the range
      1024-65536 is used for communication between trials via NCCL. This range is configured by the
      configuration parameter ``task_container_defaults.nccl_port_range`` inside ``master.yaml`` as
      described in the :ref:`cluster-configuration` guide.

   -  Two additional ephemeral TCP ports in the range 1024-65536 are used for additional intra-trial
      communication between trial containers.

   -  For TensorBoards, an inbound and outbound TCP port between 2600-2900 is used to connect the
      master and the tensorboard container.

.. _aws-cluster-configuration:

***********************
 Cluster Configuration
***********************

The Determined Cluster is configured with ``master.yaml`` file located at
``/usr/local/determined/etc/`` on the Determined master instance. You need to configure AWS dynamic
agents in each resource pool. See :ref:`cluster-configuration` for details.

**************
 Installation
**************

These instructions describe how to install Determined for the first time. For directions on how to
upgrade an existing Determined installation, see the :ref:`aws-upgrades` section below.

Ensure that you are using the most up-to-date Determined AMIs. Keep the AMI IDs handy; you will need
them later (e.g., ami-0f4677bfc3161edc8).

Master
======

To install the master, we will launch an instance from the Determined master AMI.

Let's start by navigating to the EC2 Dashboard of the AWS Console. Click "Launch Instance" and
follow the instructions below:

#. Choose AMI: find the Determined master AMI in "My AMIs" and click "Select".

#. Choose Instance Type: we recommend a t2.medium or more powerful.

#. Configure Instance: choose the ``IAM role`` according to :ref:`master-iam-role`.

#. Add Storage: click ``Add New Volume`` and add an EBS volume of at least 100GB. If you have a
   previous Determined installation that you are upgrading, you want to attach the same EBS volume
   as the previous installation. This volume will be used to store all your experiment metadata and
   checkpoints.

#. Configure Security Group: choose or create a security group according to `Set up Internet
   Access`_.

#. Review and launch the instance.

#. SSH into the Determined master and edit the config at ``/usr/local/determined/etc/master.yaml``
   according to the guide on :ref:`aws-cluster-configuration`.

#. Start the Determined master by entering ``make -C /usr/local/determined enable-master`` into the
   terminal.

Agent
=====

There is no installation needed for the agent. The Determined master will dynamically launch
Determined agent instances based on the :ref:`aws-cluster-configuration`.

.. _aws-upgrades:

**********
 Upgrades
**********

Upgrading an existing Determined installation with dynamic agents on AWS requires the same steps as
an installation without dynamic agents. See :ref:`upgrades`.

************
 Monitoring
************

Both the Determined master and agent AMIs are configured to forward system journald logs and basic
GPU metrics to AWS CloudWatch when their instances have the appropriate IAM permissions. These logs
and metrics can be helpful for diagnosing infrastructure issues when using dynamic agents on AWS.

CloudWatch Logging
==================

An instance needs the following permissions to upload logs to CloudWatch:

-  ``logs:CreateLogStream``
-  ``logs:PutLogEvents``
-  ``logs:DescribeLogStreams``

Instances will upload their logs to the log group ``/determined/determined/journald``. This log
group must be created in advance before any logs can be stored.

An example IAM policy with the appropriate permissions is below:

.. code:: json

   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": [
           "logs:CreateLogStream",
           "logs:PutLogEvents",
           "logs:DescribeLogStreams"
         ],
         "Resource": [
           "arn:aws:logs:*:*:log-group:/determined/determined/journald",
           "arn:aws:logs:*:*:log-group:/determined/determined/journald:log-stream:*"
         ]
       }
     ]
   }

CloudWatch Metrics
==================

An instance needs the following permissions to upload logs to CloudWatch:

-  ``cloudwatch:PutMetricData``

Instances will upload their metrics to namespace ``Determined``.

An example IAM policy with the appropriate permissions is below.

.. code:: json

   {
     "Version": "2012-10-17",
     "Statement": [
       {
        "Action": [
          "cloudwatch:PutMetricData"
         ],
         "Effect": "Allow",
         "Resource": "*"
       }
     ]
   }

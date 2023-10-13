.. _topic_guide_aws:

#####
 AWS
#####

This section describes how Determined runs on Amazon Web Services (AWS). For installation, see
:ref:`install-aws`.

A master node (a single, non-GPU instance) manages the cluster, provisioning and terminating agent
nodes dynamically as new workloads are started by users. The master stores metadata in an external
database; using AWS Aurora or RDS is recommended. Users interact with the cluster by using a CLI or
visiting the WebUI hosted on the master. Nodes in the cluster communicate with one another over a
Virtual Private Cloud (VPC); users interact with the master via a designated external port
configured during installation.

.. image:: /assets/images/det-cloud-architecture.png
   :alt: Diagram showing Determined Cloud Deployment Architecture on AWS

Following the diagram, a standard execution would be:

#. User submits experiment to master
#. Master creates one or more agents (depending on experiment) if they don't exist
#. Agent accesses required data, images, etc.
#. Agent completes experiment and communicates completion to master
#. Master shuts down agents that are no longer needed

This section provides details on the core resources, which are required to run Determined, and
peripheral resources, which are optionally configurable based on user requirements.

****************
 Core Resources
****************

-  **Master Node**: A single EC2 instance that:

   -  hosts the Determined WebUI where users monitor their experiments
   -  responds to commands from the Determined CLI
   -  schedules workloads
   -  manages other EC2 instances (agents) that run experiments

-  **Agent Node(s)**: For most Determined clusters in AWS, the number of agents varies with the
   volume and type of workloads currently running. All agents are managed by the master and users do
   not have to interact with them directly. For more information on scaling clusters, or dynamic
   agents, see :ref:`elastic-infrastructure`.

-  **Database**: Determined uses an Amazon Relational Database Service (Postgres) database to store
   metadata.

-  **AWS Identity and Access Management (IAM)**: IAM roles are attached to the instances to manage
   the creation of compute (EC2) resources and access to Amazon Simple Storage Service (S3) buckets
   for checkpoints, TensorBoards, and other data storage as needed.

-  **Security Groups**: VPC Security Groups ensure that each node in the cluster can communicate
   with each other.

*********************
 Periphery Resources
*********************

-  **Network/Subnetwork**: The Determined cluster runs in an existing or newly created VPC.

-  **Elastic IP**: For production clusters, the master should have an associated elastic IP;
   otherwise, AWS automatically assigns an ephemeral IP.

-  **Amazon Simple Storage Service (S3) Bucket**: The Determined cluster can leverage an existing S3
   bucket (assuming it has the correct associated permissions), or the CloudFormation script can
   create a bucket with the cluster.

.. container:: child-articles

   .. toctree::
      :glob:
      :maxdepth: 2

      ./*

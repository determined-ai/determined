.. _system-architecture:

#####################
 System Architecture
#####################

The Determined architecture comprises a single *master* and one or more *agents*. A single machine can serve as both a master and an agent.

A cluster is managed by a master node, which provisions and deprovisions agent nodes, depending on the current volume of experiments running on the cluster. When an experiment starts, the master creates agent instances and when the experiment completes, the master turns off the agents. The master also keeps all experiment metadata in a separate database, which can be queried by the user using the WebUI or CLI. All nodes in the cluster communicate internally within the cloud and you interact with the master using a port configured at installation. You do not need to interact with an agent, directly.

The following figure depicts the key functional areas of the system architecture:

.. image:: /assets/images/arch00.png

TBD: Talk about each of the boxes/connections.

The following sections describe the master and agent components in more detail.

*********
Resources
*********

Master
======

A master node is a single, non-GPU instance that manages the cluster, provisioning and terminating agent
nodes dynamically as new workloads start.

The master is the central Determined system component with the following responsibilities:

-  Stores experiment, trial, and workload metadata.
-  Schedules and dispatches experiments to agents.
-  Manages provisioning and deprovisioning of all associated agents, on-prem and in clouds.
-  Advances the experiment, trial, and workload state machines.
-  Responds to commands from your locally installed CLI.
-  Hosts the WebUI for monitoring experiments.
-  Serves the REST API.

There is typically one agent per compute server. The agent manages a number of *slots*, which are computing devices, typically a GPUs or CPUs.

Agent
=====

An agent is managed by the master and has no state. Agents communicate only with the master. Each agent has the following responsibilities:

-  Discoveres local computing devices, slots, and sends slot metadata to the master.
-  Runs the workloads at the request of the master.
-  Monitors containers and sends container information to the master.
-  Reports *trial runner* states to the master. The *trial runner* runs a trial in a containerized environment. As such, the trial runner needs access to the training data.

The volume of active experiments dictates the number of agents.

One agent can run on the master.

Database
========

A `PostgreSQL <https://www.postgresql.org/>`_ database is used to store experiment metadata.

Each Determined cluster requires access to a PostgreSQL database.
Additionally, Determined can use `Docker <https://www.docker.com/>`_ to run the master and agents.
Depending on your installation method, some of these services are installed for you:

-  On a cloud provider using ``det deploy``, Docker and PostgreSQL are preinstalled.
-  For on-premise using ``det deploy``, you need to install Docker.
-  For a manual installation, you need to install Docker and PostgreSQL.

The database commonly resides on the master.

Additional Cloud Resources
==========================

Additional resources are required for cloud environements. The following shows AWS and GCP core resource and peripheral resource requirements, for example. See the :doc:`/cluster-setup-guide/deploy-cluster/overview` for more detailed information about installing and setting up cloud environments.

Core resources
^^^^^^^^^^^^^^

-  AWS

   -  AWS Identity and Access Management (IAM)
   -  Security Groups

-  GCP

   -  Service Account
   -  Firewall Rules

Peripheral resources
^^^^^^^^^^^^^^^^^^^^

-  AWS

   -  Network/Subnetwork
   -  Elastic IP
   -  Amazon Simple Storage Service (S3) Bucket

-  GCP

   -  Network/Subnetwork
   -  Static IP
   -  Google Filestore
   -  Google Cloud Storage (GCS) bucket
   -  AWS Identity and Access Management (IAM)
   -  Security Groups

*********************
Master-Agent Workflow
*********************

The following operations represent a typical, high-level workflow:

#. Submit an experiment to the master.
#. The master creates one or more agents, depending on experiment requirements, if the agent does not already exist.
#. The agent accesses the data required to run the experiment.
#. On experiment completion, the agent communicates completion to the master.
#. The master shuts down agents that are no longer needed.

**************************************
Distributed Training - Deployment View
**************************************
TBD

**************************************
Distributed Training - Logical View
**************************************
TBD

Non-distributed Training
========================

TBD

.. image:: /assets/images/arch01.png

TBD: Talk about each of the boxes/connections.

Distributed Training
=========================

TBD

.. image:: /assets/images/arch02.png

TBD: Talk about each of the boxes/connections.

***************************************
Distributed Training - Development View
***************************************
TBD

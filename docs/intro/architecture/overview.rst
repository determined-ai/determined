.. _system-architecture:

#####################
 System Architecture
#####################

**********
Components
**********

The following figure depicts the main components of the system architecture:

.. image:: /assets/images/arch00.png

The Determined architecture comprises a single *master* and one or more *agents*. A single machine can serve as both a master and an agent.

A cluster is managed by a master node, which provisions and deprovisions agent nodes, depending on the current volume of experiments running on the cluster. When an experiment starts, the master creates agent instances and when the experiment completes, the master turns off the agents. The master also keeps all experiment metadata in a separate database, which can be queried by the user using the WebUI or CLI. All nodes in the cluster communicate internally within the cloud and you interact with the master using a port configured at installation. You do not need to interact with an agent, directly.

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

Searcher
^^^^^^^^

TBD

Metrics
^^^^^^^^

TBD

Checkpoint
^^^^^^^^^^

TBD

Scheduler
^^^^^^^^^

TBD

Agent
=====

An agent is managed by the master and has no state. Agents communicate only with the master. Each agent has the following responsibilities:

-  Discoveres local computing devices, slots, and sends slot metadata to the master.
-  Runs the workloads at the request of the master.
-  Monitors containers and sends container information to the master.
-  Reports *trial runner* states to the master. The *trial runner* runs a trial in a containerized environment. As such, the trial runner needs access to the training data.

There is typically one agent per compute server. The agent manages a number of *slots*, which are computing devices, typically a GPUs or CPUs.

The volume of active experiments dictates the number of agents.

One agent can run on the master machine.

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

Additional resources are required for cloud environments. The following shows AWS and GCP core resource and peripheral resource requirements, for example. See the :doc:`/cluster-setup-guide/getting-started` for more detailed information about installing and setting up cloud environments.

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

************************
Policies and Conventions
************************

TBD

Configuration
=============

TBD

Incrementalism
==============

Incremental features for incremental work.

Experiment Variability
======================

Variabliity is around ML models, not in how to use Determined.

Scheduling
==========

TBD

Another One
===========

TBD

*********************
Workflows
*********************

Training Implementation Workflow
================================

#. build data set
#. in each example:
#. build trial class
#. build config file that tells Det how to run experiment

   -  might change w/ different dataset

#. How do you load your data  build_training_data_loader, build_validation_data_loader

   -  how to pull the data into python

#. How do you perform training  train_batch

   objective: Find best set of parameters to get what you want. Do it repetitively to jiggle parameters

   -  loss = how well we're doing
   -  .backward & .step_optimizer = jiggling

#. How do you perform validation  evaluate_batch

   -  checks results against new data (cat image)

#. a checkpointing step

Master-Agent Workflow
=====================

#. Submit an experiment to the master.
#. The master creates one or more agents, depending on experiment requirements, if the agent does not already exist.
#. The agent accesses the data required to run the experiment.
#. On experiment completion, the agent communicates completion to the master.
#. The master shuts down agents that are no longer needed.

**************************************
Non-distributed Training
**************************************

TBD

.. image:: /assets/images/arch01.png

The Determined master launches one container, in which `entrypoint` script
in the experiment configuration is called.

The `entrypoint` script has complete freedom in how it defines and trains the
model.  The Core API is available to integrate with the rest of the Determined
platform by reporting metrics and checkpoints, checking for preemption signals,
and participating in hyperparameter searches.

**************************************
Distributed Training
**************************************

TBD

.. image:: /assets/images/arch02.png

The Determined master launches one container with multiple slots attached, or
multiple containers, each with one or more slots.  The `entrypoint` script is
called once in each container.

It is highly recommended to separate training functionality into a launcher and
a training script.  The launcher is responsible for launching multiple workers
according to the distributed training configuration, with each running the
training script.  The training script should execute the training with however
many peer workers it has available.

In fact, if both the launcher and the training script are able to deal with
non-distributed training, where the launcher launches only one worker and the
worker can operate without any peers, then switching between distributed
training and non-distributed training only requires reconfiguring
`slots_per_trial`.  This is the recommended strategy for using Determined, and
it is how Trial-based training in Determined works.

**************************************
Trial-based Training and Core API
**************************************

TBD

.. image:: /assets/images/arch03.png

Trial-based training has been available since before the Core API was
available, but can easily be thought of as a special case of Core API based
training.

With Trial-based training, you can specify just a Trial class as your
`entrypoint` rather than an entire command.  Internally, Determined launches a
Determined-provided *training script* that loads the *user trial* and starts a
Determined-provided training loop (the *trial logic*).  The training loop uses
the Core API to integrate with the rest of the Determined platform but those
details are not exposed to the user trial.

Technically non-distributed training also includes a launcher (not shown),
which starts a single worker with the *training script*.  This is an example of
the recommended strategy described in *Core API, distributed training case*.

# Distributed-training (Trial-based Training):

In Trial-based distributed training, Determined starts multiple workers with
a Determined-provided *launcher*.  Each worker runs the same *trial logic* as
before, only now training is coordinated across many workers.  The details of
distributed training are hidden as much as possible from the *user trial*.

****************
Programming View
****************

TBD

.. image:: /assets/images/arch04.png

TBD

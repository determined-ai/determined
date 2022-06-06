.. _system-architecture:

#####################
 System Architecture
#####################

This document describes the Determined platform architecture, beginning with a high-level view of the key components. This is followed by an enumeration of the policies and conventions inherent in the architecture, which are designed to support efficient distributed training implementations. Typical workflows of how the components interact are also presented, including the detailed logic for developing and integrating your code with the architectural parameters. Finally, this document presents implementation and deployment options in more detail, showing the context in which your code runs.

**********
Components
**********

The following figure represents the key, physical architectural components:

.. image:: /assets/images/arch00.png

The Determined architecture comprises a single *master* and one or more *agents* in a cluster environment.

The master node manages the cluster. At startup, an agent counts the number of resident GPUs and informs the master. The master provisions agent nodes according to the volume of experiments running on the cluster. The Determined master process on the master machine requests agents to launch containers, which applies only to Determined agent-based installations and not Kubernetes or slurm installations. When an experiment starts, the master creates agent instances and when the experiment completes, the master turns off the agent instances.

All cluster nodes communicate within the cloud. TBD: ?for what?

The master keeps experiment metadata in a PostgreSQL database, which you can query using the WebUI or CLI. 

You can interact with the master using a port configured at installation and do not need to interact with agents, directly.

The diagram shows master and agent instances on separate machines but a single machine can have both a master and one agent instances.

The following sections list key component functionality in more detail.

Master Node Functionality
=========================

A master node is a single, non-GPU instance that has the following responsibilities:

-  Store experiment, trial, and workload metadata.
-  Schedule and dispatch experiments to agents.
-  Dynamically provision and deprovision associated on-prem and cloud agents.
-  Advance experiment, trial, and workload state machines.
-  Respond to CLI commands.
-  Host the WebUI, which is primarily used to monitor experiments.
-  Serve the REST API.

Training-specific searcher, metrics, checkpointing, and scheduling functionality is also the responsibility of the master.

Searcher
^^^^^^^^

TBD

Metrics
^^^^^^^^

TBD

Checkpointing
^^^^^^^^^^^^^

TBD

Scheduling
^^^^^^^^^^

TBD

Agent Node Functionality
========================

Agent are managed by and communicate only with the master. Each agent has the following responsibilities:

-  Discover local computing devices and send device metadata to the master. A computing device is called a *slot*.
-  Run workloads at the request of the master.
-  Monitor containers and send container information to the master.
-  For a *trial runner*, which runs a trial in a containerized environment, report trial runner states to the master.

There is typically one agent per compute server. The agent manages a number of *slots*, which are computing devices, typically a GPUs or CPUs.

The volume of active experiments dictates the number of agents.

State information is not maintained on the agent. TBD: state of what?

A trial runners needs access to training data.

Database Functionality
======================

Each Determined cluster requires access to a `PostgreSQL <https://www.postgresql.org/>`_ database, which stores experiment metadata. The database typically resides on the master but is not required to.

Determined can use `Docker <https://www.docker.com/>`_ to run the master and agents. Depending on your installation method, some of these services are installed for you:

-  Using ``det deploy`` on a cloud provider, Docker and PostgreSQL are preinstalled.
-  Using ``det deploy`` on-prem, you need to install Docker.
-  For manual installation, you need to install Docker and PostgreSQL.

Additional Cloud Resources
==========================

Additional resources are required for cloud environments. The following shows AWS and GCP core resource and peripheral resource requirements:

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

Guidelines for implementing and integrating software consistent with the abstractions supported by the platform.

Configuration
=============

TBD

Incrementalism
==============

Incremental features for incremental work.

Experiment Variability
======================

Variabliity is around ML models, not in how to use Determined.

Provisioning and Deprovisioning
===============================

Something about sizing depending on volume of experiments.

Scheduling
==========

TBD

Another One
===========

TBD

*********************
Workflows
*********************

Master-Agent Workflow
=====================

#. Submit an experiment to the master.
#. If the agent does not already exist, the master creates one or more agents, depending on experiment requirements.
#. The agent accesses the data required to run the experiment.
#. On experiment completion, the agent communicates completion to the master.
#. The master shuts down agents that are no longer needed.

Training Implementation Workflow
================================

#. build data set
#. build trial class
#. build config file that tells Det how to run experiment

-  How do you load your data; how to pull the data into python: ``build_training_data_loader`` and ``build_validation_data_loader``
-  How do you perform training: ``train_batch``

   -  Find best set of parameters to get what you want.
   -  Do it repetitively to jiggle parameters

      -  loss = how well we're doing
      -  .backward & .step_optimizer = jiggling

-  How do you perform validation: ``evaluate_batch``

   -  checks results against new data (cat image)

-  checkpointing step

**************************************
Training Scenarios
**************************************

TBD

Trial-based Training Compared to using the Core API
===================================================

The following figure compares ``Trial``-based training to using the Core API directly:

.. image:: /assets/images/arch03.png

With ``Trial``-based training, you specify a ``Trial`` class as your
``entrypoint`` instead of an entire command (?python script?).  Internally, a
Determined-provided training script loads the *user trial* and starts a
Determined-provided training loop, the *trial logic*.  The training loop uses
the Core API to integrate with the rest of the Determined platform but those
details are not exposed to the user trial. Trial-based training can be viewed as a special case of Core API training.

In Trial-based distributed training, Determined starts multiple workers with
a Determined-provided *launcher*.  Each worker runs the same trial logic coordinated across many workers. The distributed training details are hidden as much as possible from the user trial.

So in the Trial-based training case, the entrypoint script is always a launcher (if the user specifies the legacy trial class format of entrypoint in their config, we just automatically convert it).  The launcher does basically nothing in non-distributed training, and it launches one worker.  That one worker looks just like it does on the left.The second diagram, which depicts non-distributed training, has a rather meaningless "training script" label, and its content appears to be worker from the first diagram.  That's because in non-distributed training, the launcher does nothing, and it's easy enough to describe it to users as if it doesn't exist.The third diagram is just like the second diagram, also depicting non-distributed training.  It also elides the launcher-that-does-nothing from the explanation.  The goal of the third diagram was to contrast with the fourth diagram; the dotted boundary shows what the user controls.  In the Trial-based training, they write a little plugin (the Trial) that fits into a framework we define (the Training Loop or Trial Logic... same thing).  In that framework we define, the TrainingLoop/TrialLogic uses the Core API on the users' behalf.The fourth diagram has a dotted line around the whole entrypoint script (the label I gave it was "training script" but I think that was imprecise, as I noted above).  The point was to show that in Core API-based training, there  is no framework or plugins, you just do what you want and interact with the core api directly.

Alternatively, you can call Core API methods directly. The difference implied for your code are described in the following sections about non-distributed and distributed Core API implementations.

Non-distributed Training using Core API
=======================================

The Core API enables you to integrate directly with the the Determined
platform by:

-  reporting metrics and checkpoints
-  checking for preemption signals
-  participating in hyperparameter searches

The following figure shows the software logic you need to provide when using Core API, directly:

.. image:: /assets/images/arch01.png

The Determined master launches one container, which calls the *training script* specified in the experiment configuration file. The launcher, which is not shown, starts a single worker with the training script.

The training script has complete freedom in how it defines and trains the model.

Distributed Training using Core API
===================================

TBD

.. image:: /assets/images/arch02.png

The Determined master launches one container with multiple slots attached, or
multiple containers, each container with one or more slots.  The training script is
called once in each container.

It is highly recommended to separate training functionality into a launcher and
a training script.  The launcher is responsible for launching multiple workers
according to the distributed training configuration, with each running the
training script.  The training script should execute training with the number of available peer workers.

If both the launcher and the training script are able to handle
non-distributed training, where the launcher launches only one worker and the
worker can operate without any peers, switching between distributed
training and non-distributed training only requires changing the
``slots_per_trial`` configuration parameter.  This is the recommended strategy for using Determined, and
it is how Trial-based training works.

****************
Programming View
****************

TBD

.. image:: /assets/images/arch04.png

TBD

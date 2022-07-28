.. _det-system-architecture:

.. _system-architecture:

#####################
 System Architecture
#####################

.. pull-quote::

   "Determined makes it easy to transform data into probabilities."

   -- Garrett Goon, 2022

**********
 Overview
**********

This document describes the Determined system architecture in terms of its components and behavior,
providing context for both system administrators and machine learning engineers before exploring
Determined in more detail.

Determined implements the training workflow shown in the **Run Training** section of the following
figure:

.. image:: /assets/images/arch12.png

**TBD: Describe what the relevant boxes are and how they interact; e.g., built-in support for data,
checkpointing, and metric storage.**

The following figure is a more detailed view of the Determined platform:

.. image:: /assets/images/arch11.png

**TBD: Describe what the boxes are and how they interact. And, of course, generalize the diagram.**

************
 Components
************

Determined deployment can range from a local laptop and on-premises setups to distributed master and
agent nodes, all with database and file storage connectivity, containerization, and cloud resource
options.

The following figure shows the key master and agent instances on separate machines:

.. image:: /assets/images/arch00.png

However, a single machine can have both a master and one agent instance.

Master and Agent Nodes
======================

Determined comprises a single *master* and one or more *agents* in a cluster environment. The master
is a single, non-GPU node that manages the cluster and keeps experiment metadata in a PostgreSQL
database

You can interact directly with the master using the WebUI or CLI. The master alone communicates with
agents as needed.

Master Node Scope of Responsibility
-----------------------------------

Master node responsibilities:

-  Stores experiment, trial, and workload metadata.
-  Schedules and dispatch experiments to agents.
-  Dynamically provisions and deprovisions on-premises and cloud agents.
-  Advances experiment, trial, and workload state machines.
-  Responds to CLI commands.
-  Hosts the WebUI, which is primarily used to monitor experiments.
-  Serves the REST API.

The master is also responsible for the following training-specific searcher, metrics, checkpointing,
and scheduling functionality:

+---------------+----------------------------------------------------------------------+
| Function      | Description                                                          |
+===============+======================================================================+
| Searcher      | The *Searcher* implements a hyperparameter search algorithm and is   |
|               | responsible for coordinating the work of all experiment trials.      |
+---------------+----------------------------------------------------------------------+
| Metrics       | *Metrics* provides persistent storage of metrics reported by Trials. |
+---------------+----------------------------------------------------------------------+
| Checkpointing | *Checkpointing* captures the training state at a given time. You can |
|               | choose to save checkpoint information in the *model registry*.       |
+---------------+----------------------------------------------------------------------+
| Scheduling    | *Scheduling* schedules jobs to run, ensuring that all of the compute |
|               | resources required for a job are available before the job launches.  |
+---------------+----------------------------------------------------------------------+

Agent Node Scope of Responsibility
----------------------------------

Agent node responsibilities:

-  Discovers local computing devices and sends device/slot metadata to the master.
-  Starts task containers at the request of the master.
-  Monitors task containers and sends container information to the master.

The agent manages a number of GPU or CPU devices, which are referred to as *slots*. There is
typically one agent per compute server and the active experiment volume dictates the number of
agents needed.

Agents communicate only with the master and do not persist any information across restarts.

PostgreSQL Database
===================

Each cluster requires access to a `PostgreSQL <https://www.postgresql.org/>`_ database to store
experiment and trial metadata. Although not required, the database typically resides on the master.
When you use the ``det deploy`` command, Determined prepares a PostgreSQL instance for you.
Otherwise, you need to manually install PostgreSQL.

Docker Images
=============

Determined launches workloads using `Docker <https://www.docker.com/>`_ containers. Determined
provides a default container that includes common deep learning libraries and frameworks.

If you use the ``det deploy aws`` or ``det deploy gcp`` command on a cloud provider, Docker is
preinstalled. For a manual or on-premises deployment using ``det deploy local``, you need to
manually install Docker.

*******************
 Design Principles
*******************

The Determined platform is implemented according to the following principles.

Concurrency
===========

Determined provides three types of concurrent processing that take advantage of a multi-GPU
environment:

-  *Parallelism across experiments.* Determined can schedule multiple experiments to run
   concurrently across the available GPUs.

-  *Parallelism within an experiment.* Determined can schedule multiple experiment trials. A
   hyperparameter search can train multiple trials simultaneously, each on a different GPU.

-  *Parallelism within a trial.* Determined can use multiple GPUs to speed up training of a single
   trial. Determined coordinates across multiple GPUs on a single machine, or across multiple GPUs
   on multiple machines, to improve single-trial training performance.

Reproducibility
===============

Determined supports reproducible machine learning experiments to ensure that Determined experiments
are deterministic. Rerunning a previous experiment is expected to produce an identical model. This
ensures that if the model produced from an experiment is ever lost, it can be recovered by rerunning
the experiment that produced it.

Determined controls and reproduces the following sources of randomness:

-  Hyperparameter sampling decisions.
-  Initial weights for a given hyperparameter configuration.
-  Shuffling of training data in a trial.
-  Dropout or other random layers.

Determined does not currently support controlling non-determinism in floating-point operations.

Configuration
=============

Determined is a deep learning training platform that simplifies infrastructure management for domain
experts while enabling configuration-based deep learning functionality. Configuration files control
the operation and behavior of:

-  Master nodes
-  Agent nodes
-  Experiments
-  Jobs

You can use *configuration templates* to share experiment configurations within an organization.

Provisioning and Deprovisioning
===============================

A cluster is managed by the master, which provisions and deprovisions agents depending on the
current volume of experiments on the cluster.

Scheduling
==========

The master schedules distributed training jobs automatically, ensuring that all of the compute
resources required for a job are available before the job is launched.

Job queue management is available to the fair share, priority, and Kubernetes preemption schedulers
and exposes scheduler functionality for visibility and control over scheduling decisions. The *job
queue* provides information about job ordering and which jobs are queued, which you can manage
dynamically.

********************
 Training Scenarios
********************

You have the option of using trial-based training or accessing Core API directly to run your
training logic. Trial-based training hooks into the Determined framework to run the training loop,
while Core API-based training does not hook into the framework.

The following figure shows the difference between ``Trial``-based training and using the Core API
directly, from a programming perspective:

.. image:: /assets/images/arch03.png

You run an experiment by specifying a *launcher*. The distributed training launcher must implement
the following logic:

-  Launch all of the workers you want, passing any required peer info, such as rank or chief IP
   address, to each worker.
-  Monitor workers and handle worker termination.

Launcher options:

-  legacy bare-Trial-class

   In general, you convert existing training code by subclassing a ``Trial`` class and implementing
   methods that advertise components of your model, such as model architecture, data loader,
   optimizer, learning rate scheduler, and callbacks. Your ``Trial`` class inherits from Determined
   classes provided for PyTorch, PyTorch Lightning, Keras, or Estimator, depending on your
   framework. This is called the trial definition and by structuring your code in this way,
   Determined can run the training loop, providing advanced training and model management
   capabilities.

-  Determined predefined launchers:

   +---------------------+-------------------------------------------------------------------+
   | Launcher            | Description                                                       |
   +=====================+===================================================================+
   | Horovod             | The Horovod launcher is a wrapper around `horovodrun              |
   |                     | <https://horovod.readthedocs.io/en/stable/summary_include.html>`_ |
   |                     | which automatically configures the workers for the trial.         |
   +---------------------+-------------------------------------------------------------------+
   | PyTorch Distributed | The PyTorch launcher is a Determined wrapper around the           |
   |                     | ``torch.distributed.run`` PyTorch native distributed training     |
   |                     | launcher.                                                         |
   +---------------------+-------------------------------------------------------------------+
   | DeepSpeed           | The DeepSpeed launcher launches a training script under           |
   |                     | ``deepspeed`` with automatic IP address, sshd container, and      |
   |                     | shutdown handling.                                                |
   +---------------------+-------------------------------------------------------------------+

-  A custom launcher.

-  A command with arguments, which runs in a container.

Trial-based Distributed Training
================================

In trial-based distributed training, Determined starts multiple workers with a Determined-provided
*launcher*. With trial-based training, you specify a ``Trial`` class as your entry point. A
Determined-provided training script loads the *user trial* and starts a Determined-provided *trial
logic* training loop. The training loop makes Core API calls on your behalf. Each worker runs the
same trial logic, which is coordinated across multiple workers.

Non-distributed Training using the Core API
===========================================

In Core API-based training, you interact directly with the Determined platform to:

-  report metrics and checkpoints
-  check for preemption signals
-  run hyperparameter searches

The following figure shows the logic you need to provide when you use the Core API, directly:

.. image:: /assets/images/arch01.png

The Determined master launches a single container, which calls the *training script* specified in
the experiment configuration file. The launcher starts a single worker using the training script.
The training script has full flexibility in how it defines and trains a model.

Distributed Training using the Core API
=======================================

The following figure shows multiple agents in a distributed training scenario using the Core API:

.. image:: /assets/images/arch02.png

The master launches a single container with multiple *slots* attached or multiple containers that
each have one or more slots. The training script is called once in each container.

The launcher is responsible for launching multiple workers according to the distributed training
configuration, with each worker running the training script. The training script should execute
training with the number of available peer workers. These should be implemented in separate launcher
and training scripts.

If both the launcher and the training script are able to handle non-distributed training, where the
launcher launches only one worker and the worker can operate without peer workers, switching between
distributed training and non-distributed training requires only changing the ``slots_per_trial``
configuration parameter. This is the recommended strategy for using Determined and is how
trial-based training works.

*******************
 Training Workflow
*******************

The training workflow generally involves:

-  saving your training data sets in an accessible location.
-  writing training code to download and train a model using Determined APIs.
-  submitting an experiment to run the training code on available resources.

Set up Training
===============

#. Create training and validation datasets.

   -  The training dataset is a large dataset used to update the model and is the set you train on.
   -  The validation dataset is a distinct dataset used to compare the trained model against. You
      stop training when performance metrics begin to diverge.

#. Save your dataset.

   Data plays a fundamental role in machine learning model development. The best way to load data
   into your ML models depends on several factors, including whether you are running on-premise or
   in the cloud, the size of your datasets, and your security requirements. Determined supports the
   following methods for accessing your dataset:

   -  Uploaded the dataset as part of the experiment directory, which usually includes your training
      API implementation. Determined injects the contents of the experiment directory into each
      trial container that is launched for the experiment. Any file in the directory can then be
      accessed by your model code.

   -  Use a distributed file system to store data, which enables a cluster of machines to access a
      shared dataset using the POSIX file system interface.

   -  Use object stores to manage data as a collection of key-value pairs. Object storage is
      particularly popular in cloud environments.

Define a Training Loop
======================

After initialization, every worker runs the following, general training loop, repeatedly:

#. Perform a forward and backward pass on a unique *batch* subset of data and generate a set of
   updates to the model parameters based on the processed data.
#. Communicate updates to other workers so that all workers see all of the updates made during that
   batch.
#. Average the updates by the number of workers and apply the updates to its copy of the model
   parameters. This results in identical solution states for all workers.

You code the model architecture to define what to do with the data. When you use a ``Trial`` class
for training, the ``Trial`` class handles the Core API entirely but you need access to the
underlying framework to build your model and dataset, directly using PyTorch or TensorFlow for
example. The following figure shows the relationship of user code to ``PyTorchTrial`` and supported
frameworks:

.. image:: /assets/images/arch09.png

When you use ``PyTorchTrial``, you use PyTorch or TensorFlow to define the model, dataset,
optimizer, and other trial-specific objects. ``PyTorchTrial`` handles both the Core API details and
the PyTorch or TensorFlow details needed to run the training loop.

Programming steps:

#. Create an Experiment, which involves the following activities:

   -  Initializing Objects Optimization Step/Using Optimizer Using Learning Rate Scheduler

      **TBD: need to decode this**

   -  Build a dataset.

   -  Build a ``Trial`` class.

   -  Build a configuration file that describes how to run the experiment.

   -  Specify where your data is located and how to load the data, or how to pull the datasets into
      python:

      The ``build_training_data_loader`` and ``build_validation_data_loader`` methods efficiently
      feed data into the model and can include additional data processing steps.

   -  Specify how to perform training. The ``train_batch`` method uses all the PyTorch machinery
      through the PyTorchTrial API, coordinating all actions including scheduling.

      The objective is to find the best set of parameters to use. You train on your dataset
      repetitively with the backward pass and step optimizer,
      ``self.context.step_optimizer(self.optimizer)``. The *loss*,
      ``self.context.step_optimizer(self.optimizer)``, at each iteration tells how well training is
      performing.

#. Define the validation loop, using the ``evaluate_batch()`` method to validate your model. You
   might also check results against new data.

#. Configure a launcher as your processing entry point. The launcher specification can take one of
   the following forms:

   -  An arbitrary entry point script name.
   -  The name of a preconfigured launch module and script name.
   -  The name of a preconfigured launcher and legacy ``Trial`` class specification.
   -  A legacy ``Trial`` class specification.

Submit an Experiment
====================

After preparing your dataset and coding your model, submit an experiment, which involves the
following activities:

#. Submit an experiment to the master. If the agent does not already exist, the master provisions
   agent nodes according to the volume of experiments. When an experiment starts, the master creates
   agent instances.

   -  Each agent notifies the master of the number of resident GPUs.
   -  For agent-based installations, excluding `Kubernetes <https://kubernetes.io/>`_ and `Slurm
      <https://www.schedmd.com/>`_, the master process requests agents to launch containers.

#. The agent downloads and loads the data specified for the experiment.

#. On experiment completion, the agent communicates completion to the master.

#. The master shuts down agents that are no longer needed.

Scheduler
=========

The *scheduler* decides which jobs are allocated time on the scheduler and can preempt running jobs.
Preemption can occur if a higher-priority job arrives or because of user actions, such as clicking
the WebUI pause button.

Preemption is participatory, so running jobs save a checkpoint state and shut down cleanly. If you
do not preempt the job, your code runs to completion.

Checkpointing
=============

A *checkpoint* contains the training state at a point in time. Checkpoints are key to persisting
your trained model after training completes by providing the ability to pause and continue training
without losing progress. The master stores metadata about each checkpoint in external storage.

A checkpoint includes the model definition Python source code, experiment configuration file,
network architecture, and the model parameter values and hyperparameters. When using a stateful
optimizer during training, checkpoints also include the optimizer or learning rate state. You can
also embed arbitrary metadata in checkpoints

The *model registry* is a way to group together conceptually related checkpoints, including
checkpoints across different experiments, storing metadata and long-form notes about a model, and
retrieving the latest version of a model for use or further development. The model registry can be
accessed using the WebUI, Python API, REST API, or CLI.

********************
 Using the Core API
********************

When you use Core API directly, you can train using the framework of your choice, and you use the
**TBD**. The following figure shows that your code has direct access to the Core API and supported
frameworks:

.. image:: /assets/images/arch10.png

The Core API exposes mechanisms to integrate your code with the Determined platform. Each
``core_context`` component corresponds to a Determined platform component, as described in the
following sections.

.. image:: /assets/images/arch04.png

The ClusterInfo API provides the master with information about the currently-running task and is
available only to tasks running on the cluster. ``ClusterInfo`` exposes properties that are set for
tasks while running on the cluster, such as ``container_addrs``, which contains the IP addresses of
all containers participating in a distributed task. The ClusterInfo API is intended to be most
useful when implementing custom launchers.

The following describes the Core API interfaces in more detail.

Metrics
=======

The master *metrics* storage is the persistent storage of metrics reported by all trials. WebUI
graphs are rendered from data in this store. Operations such as **top-N checkpoints** read metrics
storage to find which checkpoints correspond to the best searcher metric.

The ``core_context.train`` component reports metrics to be stored in metric stroage, using
``.report_training_metrics()`` or ``.report_validation_metrics()``.

Searcher
========

There is a single *searcher* for each experiment, which implements a hyperparameter search algorithm
and is responsible for coordinating the work of all of the trials in an experiment.

The ``core_context.searcher`` component enables code to integrate with the searcher for an
experiment. You can use the ``core_context.searcher`` class for your trial to participate in the
hyperparameter search for an experiment.

The role of each trial in the hyperparameter search is to iterate through the ``SearcherOperation``
objects from the ``core_context.searcher.operations`` method. Each ``SearcherOperation`` has a
``.length`` that describes how long the trial should train. The trial evaluates the searcher metric
at that point and reports the metric using the ``op.report_completed(metric_value)`` method.

Optionally, each trial can report training progress using the ``op.report_progress`` method. The
searcher collects all reported progress from all trials in the experiment and displays the
aggregated progress in the WebUI.

Checkpoint
==========

**TBD: The programming view diagram is missing a Checkpoint Storage block, which is outside of the
Determined-master.**

The ``core_context.checkpoint`` component is used to upload and download checkpoint contents from
checkpoint storage and to fetch and store metadata from the master. The ``upload()`` method takes a
directory to upload to external storage with the checkpoint metadata you want to set with the
master. You can fetch the metadata using the ``get_metadata()`` method and the file contents using
the ``download()`` method.

Scheduler
=========

The ``core_context.preempt`` component can be used to preempt training by periodically calling the
``.should_preempt()`` method and taking appropriate action, such as saving a checkpoint and exiting
if it indicates that your job is preempted.

**********
 See Also
**********

Setup:

-  :doc:`/cluster-setup-guide/basic`
-  :doc:`/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-prem/overview`
-  :doc:`/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-aws/overview`
-  :doc:`/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-gcp/overview`

Training:

-  :doc:`/training/setup-guide/overview`
-  :doc:`/training/dtrain-introduction`

Interface:

-  :doc:`/interfaces/commands-and-shells`
-  :doc:`/interfaces/notebooks`

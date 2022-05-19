.. _features:

##########
 Concepts
##########

Deep learning practitioners come from a variety of disciplines. Depending on their background, some
practitioners have strong foundations in engineering, while others focus on statistics and domain
expertise. Determined AI is a deep learning training platform that simplifies infrastructure
management for domain experts while enabling configuration-based deep learning functionality that is
generally inconvenient to implement for engineering-oriented practitioners.

Many current systems are point solutions for specific problems in deep learning, so combining the
systems is tough and inefficient. Determined's cohesive end-to-end training platform provides
best-in-class functionality for deep learning model training, with a suite of benefits, including:

-  **Cluster management**: Automatically manage ML accelerators (e.g., GPUs) on-premise or in cloud
   VMs, using your own environment that automatically scales for your on-demand workloads.
   Determined runs in either AWS or GCP, so you can switch easily as your needs require. See
   :doc:`/concepts/resource-pool`, :doc:`/concepts/scheduling`, and
   :doc:`/concepts/elastic-infrastructure`.

-  **Containerization**: Develop and train models in customizable containers, which enable simple
   and consistent dependency management throughout the model development lifecycle. See
   :doc:`/prepare-environment/custom-env`.

-  **Cluster-backed notebooks, commands, and shells**: Leverage your shared cluster computing
   devices in a more versatile environment. See :doc:`/features/notebooks` and
   :doc:`/features/commands-and-shells`.

-  **Experiment collaboration**: Automatically track the configuration and environment for each of
   your experiments, facilitating reproducibility and collaboration among teams. See
   :doc:`/training-run/index`.

-  **Visualization**: Visualize your model and training procedure by using Determined's built-in
   WebUI, and also by launching managed :doc:`/features/tensorboard` instances.

-  **Fault tolerance**: Models are checkpointed throughout the training process and can be restarted
   from the latest checkpoint automatically. This enables training jobs to automatically tolerate
   transient hardware or system issues in the cluster.

-  **Automated model tuning**: Optimize models by searching through conventional hyperparameters or
   macro-architectures, using a variety of search algorithms. Hyperparameter searches are
   automatically parallelized across the accelerators in the cluster. See
   :doc:`/training-hyperparameter/index`.

-  **Distributed training**: Easily distribute a single training job across multiple accelerators to
   speed up model training and reduce model development iteration time. Determined uses synchronous,
   data-parallel distributed training, with key performance optimizations over other available
   options. See :doc:`/training-distributed/index`.

-  **Broad framework support**: Leverage these capabilities using any of the leading machine
   learning frameworks without having to manage a different cluster for each. Different frameworks
   for different models can be used without worrying about future lock-in. See
   :doc:`/training-apis/index`.

.. _det-system-architecture:

*********************
 System Architecture
*********************

Determined consists of a single **master** and one or more **agents**. There is typically one agent
per compute server; a single machine can serve as both a master and an agent.

The **master** is the central component of the Determined system. It is responsible for

-  Storing experiment, trial, and workload metadata.
-  Scheduling and dispatching work to agents.
-  Managing provisioning and deprovisioning of agents in clouds.
-  Advancing the experiment, trial, and workload state machines over time.
-  Hosting the WebUI and the REST API.

An **agent** manages a number of **slots**, which are computing devices (typically a GPU or CPU). An
agent has no state and only communicates with the master. Each agent is responsible for

-  Discovering local computing devices (slots) and sending metadata about them to the master.
-  Running the workloads that are requested by the master.
-  Monitoring containers and sending information about them to the master.

The **trial runner** runs a trial in a containerized environment. So the trial runners are expected
to have access to the data that will be used in training. The **agents** are responsible for
reporting the states of **trial runner** to the master.

**********
 Training
**********

.. _concept-experiment:

We use **experiments** to represent the basic unit of running the model training code. An experiment
is a collection of one or more trials that are exploring a user-defined hyperparameter space. For
example, during a learning rate hyperparameter search, an experiment might consist of three trials
with learning rates of .001, .01, and .1.

.. _concept-trial:

A **trial** is a training task with a defined set of hyperparameters. A common degenerate case is an
experiment with a single trial, which corresponds to training a single deep learning model.

In order to run experiments, you need to write your model training code. We use **model definition**
to represent a specification of a deep learning model and its training procedure. It contains
training code that implements :doc:`training APIs </training-apis/index>`.

For each experiment, you can configure a **searcher**, also known as a **search algorithm**. The
search algorithm determines how many trials will be run for a particular experiment and how the
hyperparameters will be set. More information can be found at :doc:`/training-hyperparameter/index`.

.. toctree::
   :maxdepth: 1
   :hidden:

   elastic-infrastructure
   resource-pool
   scheduling
   yaml

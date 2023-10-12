.. _introduction-determined:

########
Features
########

.. _features:

Determined is an all-in-one deep learning platform, compatible with PyTorch and TensorFlow.

It takes care of:

-  Distributed training for faster results.
-  Hyperparameter tuning for obtaining the best models.
-  Resource management for cutting cloud GPU costs.
-  Experiment tracking for analysis and reproducibility.

Configurable Interactive Jobs
=============================

The behavior of interactive jobs, such as :ref:`TensorBoards <tensorboards>`, :ref:`notebooks
<notebooks>`, :ref:`commands, and shells <commands-and-shells>`, can be influenced by setting a
variety of configuration variables. These configuration variables are similar but not identical to
the configuration options supported by :ref:`experiments <experiment-config-reference>`.

Configuration settings can be specified by passing a YAML configuration file when launching the
workload via the Determined CLI.

Configuration variables can also be set directly on the command line when any Determined task,
except a TensorBoard, is launched.

Options set via ``--config`` take precedence over values specified in the configuration file.
Configuration settings are compatible with any Determined task unless otherwise specified.

Determined CLI
==============

One of the key components of the Determined platform is the :ref:`command-line interface (CLI)
<cli-ug>`. The CLI serves as a primary entry point for interacting with Determined, providing a way
to efficiently manage and control various aspects of the system. The following list describes some
of the tasks you can perform with the Determined CLI:

-  Experiment management: Running experiments is a fundamental part of the machine learning process.
   With the CLI, you can effortlessly create, list, and manage experiments, as well as access
   important experiment metrics and logs.

-  Queue management: The CLI enables users to manage their job queues, monitor the progress of
   ongoing tasks, and even prioritize or cancel jobs as needed.

-  Notebook management: Jupyter notebooks are an essential tool for data scientists and machine
   learning engineers. The CLI simplifies the process of creating, launching, and managing Jupyter
   notebooks within the platform.

-  TensorBoard integration: TensorBoard is a popular visualization tool for TensorFlow projects. The
   CLI allows users to easily launch and manage TensorBoard instances, making it simple to visualize
   and analyze the training progress of their models.

Commands and Shells
===================

:ref:`Commands and shells <commands-and-shells>` support free-form tasks.

In Determined, a developer uses an :ref:`experiment <experiments>`, to run a trial. Outside of
trials, a developer can use the ``det cmd`` Command (the capitalization indicates it is a specific
feature of Determined). This Command facilitates the execution of a user-defined program on the
cluster. On the other hand, shells initiate SSH servers, enabling the interactive use of cluster
resources.

Commands and shells enable developers to use a Determined cluster and its GPUs without having to
write code conforming to the trial APIs. Commands are useful for running existing code in a batch
manner; shells provide access to the cluster in the form of interactive `SSH
<https://en.wikipedia.org/wiki/SSH_(Secure_Shell)>`_ sessions.

Configuration Templates
=======================

At a typical organization, many Determined configuration files will contain similar settings. For
example, all of the training workloads run at a given organization might use the same checkpoint
storage configuration. One way to reduce this redundancy is to use *configuration templates*. With
this feature, you can move settings that are shared by many experiments into a single YAML file that
can then be referenced by configurations that require those settings.

Each configuration template has a unique name and is stored by the Determined master. If a
configuration specifies a template, the effective configuration of the task will be the result of
merging the two YAML files (configuration file and template). The semantics of this merge operation
is described under :ref:`Configuration Templates: Merge Behavior <config-templates-merge-behavior>`.
Determined stores this expanded configuration so that future changes to a template will not affect
the reproducibility of experiments that used a previous version of the configuration template.

A single configuration file can use at most one configuration template. A configuration template
cannot use another configuration template.

Queue Management
================

The Determined Queue Management system extends scheduler functionality to offer better visibility
and control over scheduling decisions. It does this using the Job Queue, which provides better
information about job ordering, such as which jobs are queued, and permits dynamic job modification.

Queue Management is a new feature that is available to the fair share scheduler and the priority
scheduler. Queue Management, described in detail in the following sections, shows all submitted jobs
and their states and lets you modify some configuration options, such as priority, position in the
queue, and resource pool.

To begin managing job queues, go to the WebUI ``Job Queue`` section or use the ``det job`` set of
CLI commands.

Model Registry
==============

The Model Registry is a way to group conceptually related checkpoints (including ones across
different experiments), store metadata and long-form notes about a model, and retrieve the latest
version of a model for use or further development. The Model Registry can be accessed through the
WebUI, Python SDK, REST API, or CLI, though the WebUI has some features that the others are missing.

The Model Registry is a top-level option in the navigation bar. This will take you to a page listing
all of the models that currently exist in the registry, and allow you to create new models. You can
select any of the existing models to go to the Model Details page, where you can view and edit
detailed information about the model. There will also be a list of every version associated with the
selected model, and you can go to the Version Details page to view and edit that version's
information.

For more information about how to use the model registry, see `Organizing Models in the Model
Registry <../post-training/model-registry.html>`_

Notebooks
=========

`Jupyter Notebooks <https://jupyter.org/>`__ are a convenient way to develop and debug machine
learning models, visualize the behavior of trained models, or even manage the training lifecycle of
a model manually. Determined makes it easy to launch and manage notebooks.

Determined Notebooks have the following benefits:

-  Jupyter Notebooks run in containerized environments on the cluster. We can easily manage
   dependencies using images and virtual environments. The HTTP requests are passed through the
   master proxy from and to the container.

-  Jupyter Notebooks are automatically terminated if they are idle for a configurable duration to
   release resources. A notebook instance is considered to be idle if it is not receiving any HTTP
   traffic and it is not otherwise active (as defined by the ``notebook_idle_type`` option in the
   :ref:`task configuration <command-notebook-configuration>`).

.. note::

   -  Once a Notebook is terminated, it is not possible to restore the files that are not stored in
      the persistent directories. **You need to ensure that the cluster is configured to mount
      persistent directories into the container and save files in the persistent directories in the
      container.** See :ref:`notebook-state` for more information.

   -  If you open a Notebook tab in JupyterLab, it will automatically open a kernel that will not be
      shut down automatically so you need to manually terminate the kernels.

TensorBoards
============

`TensorBoard <https://www.tensorflow.org/tensorboard>`__ is a widely used tool for visualizing and
inspecting deep learning models. Determined makes it easy to use TensorBoard to examine a single
experiment or to compare multiple experiments.

TensorBoard instances can be launched via the WebUI or the CLI. To launch TensorBoard instances from
the CLI, first :ref:`install the CLI <install-cli>` on your development machine.

Workspaces and Projects
=======================

**Workspaces** and **projects** provide a way to organize experiments. A project is a collection of
experiments, and a workspace is a collection of projects. Learn more about workspaces and projects
at :ref:`workspaces`.
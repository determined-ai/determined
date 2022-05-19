.. _experiments:

###################
 Run Training Code
###################

This section covers how to run training code by submitting your code to a cluster and running it as
an experiment.

You run an experiment by providing a *launcher* and specify the launcher in the experiment
configuration ``endpoint`` field. Launcher options are:

-  legacy bare-Trial-class

-  Determined predefined launchers:

   -  `Horovod Launcher`_
   -  `PyTorch Distributed Launcher`_
   -  `DeepSpeed Launcher`_

-  custom launcher or use one of the Determined predefined launchers

-  a command with arguments, which is run in the container

For distributed training, it is good practice to separate the launcher that starts a number of
distributed workers from your training script, which typically runs each worker. The distributed
training launcher must implement the following logic:

-  Launch all of the workers you want, passing any required peer info, such as rank or chief ip
   address, to each worker.
-  Monitor workers. If a worker exits with non-zero, the launcher should terminate the remaining
   workers and exit with non-zero.
-  Exit zero after all workers exit zero.

These requirements ensure that distributed training jobs do not hang after a single worker failure.

**********************
 Configure a Launcher
**********************

The entry point of a model is configured in the :ref:`Experiment Configuration
<experiment-configuration>` file and specifies the path of the model and how it should be launched.
Predefined launchers are provided in the ``determined.launch`` module and custom scripts are also
supported.

The launcher is configurable in the experiment configuration ``entrypoint`` field. The
``entrypoint`` trial object or script launches the training code.

.. code:: yaml

   entrypoint: python3 -m (LAUNCHER) (TRIAL_DEFINITION)

or a custom script:

.. code:: yaml

   entrypoint: python3 script.py arg1 arg2

Preconfigured launcher ``entrypoint`` arguments can differ but have the same format:

.. code:: bash

   python3 -m (LAUNCH_MODULE) (--trial TRIAL)|(SCRIPT...)

where ``(LAUNCH_MODULE)`` is a Determined launcher and ``(--trial TRIAL)|(SCRIPT...)`` refers to the
training script, which can be in a simplified format that the Determined launcher recognizes or a
custom script.

You can write a custom launcher, in which case the launcher should wrap each rank worker in the
``python3 -m determined.launch.wrap_rank $RANK CMD [ARGS...]`` script, so the final logs can be
separated according to rank in the WebUI.

Training Code Definition
========================

To launch a model or training script, either pass a trial class path to --trial or run a custom
script that runs the training code. Only one of these can be used at the same time.

Trial Class
-----------

.. code:: bash

   --trial TRIAL

To specify a trial class to be trained, the launcher accepts a TRIAL argument in the following
format:

.. code:: yaml

   filepath:ClassName

where filepath is the location of your training class file, and ClassName is the name of the Python
training class

Custom Script
-------------

A custom script can be launched under a supported launcher instead of a trial class definition, with
arguments passed as expected.

Example Python script command:

.. code:: bash

   script.py [args...]

Horovod Launcher
================

Format:

``determined.launch.horovod [[HVD_OVERRIDES...] --] (--trial TRIAL)|(SCRIPT...)``

The horovod launcher is a wrapper around `horovodrun
<https://horovod.readthedocs.io/en/stable/summary_include.html#running-horovod>`_ which
automatically configures the workers for the trial. You can pass arguments directly to
``horovodrun``, overriding Determined values, as ``HVD_OVERRIDES``, which must end with a ``--`` to
separate the overrides from the normal arguments.

Example:

.. code:: bash

   python3 -m determined.launch.horovod --fusion-threshold-mb 1 --cycle-time-ms 2 -- --trial model_def:MyTrial

PyTorch Distributed Launcher
============================

Format:

``determined.launch.torch_distributed [[TORCH_OVERRIDES...] --] (--trial TRIAL)|(SCRIPT...)``

This launcher is a Determined wrapper around PyTorch's native distributed training launcher,
torch.distributed.run. Any arbitrary override arguments to torch.distributed.run are accepted, which
overrides default values set by Determined. See the official PyTorch documentation for information
about how to use ``torch.distributed.run``. The optional override arguments must end with a ``--``
separator before the trial specification.

Example:

.. code:: bash

   python3 -m determined.launch.torch_distributed --rdzv_endpoint=$CUSTOM_RDZV_ADDR -- --trial model_def:MyTrial

DeepSpeed Launcher
==================

Format:

``determined.launch.deepspeed [[DEEPSPEED_ARGS...] --] (--trial TRIAL)|(SCRIPT...)``

The DeepSpeed launcher launches a training script under ``deepspeed`` with automatic handling of:

-  IP addresses
-  sshd containers
-  shutdown

See the DeepSpeed `Launching DeepSpeed Training
<https://www.deepspeed.ai/getting-started/#launching-deepspeed-training>`_ documentation for
information about how to use the DeepSpeed launcher.

Example:

.. code:: bash

   python3 -m determined.launch.deepspeed --trial model_def:MyTrial

Use the help option to get the latest usage:

.. code:: bash

   python3 -m determined.launch.deepspeed -h

Legacy Launcher
===============

Format:

``entrypoint: model_def:TrialClass``

The entry point field expects a predefined or custom script, but also supports legacy file and trial
class definitions.

When you specify a trial class as the entry point, it must be a subclass of a Determined trial
class.

Each trial class is designed to support one deep learning application framework. When training or
validating models, the trial might need to load data from an external source so the training code
needs to define data loaders.

A TrialClass is located in the ``model_def`` filepath and launched automatically. This is considered
legacy behavior. By default, this configuration automatically detects distributed training, based on
slot size and the number of machines, and launches with Horovod for distributed training. If used in
a distributed training context, the entry point is:

.. code:: bash

   python3 -m determined.launch.horovod --trial model_def:TrialClass

Nested Launchers
================

The entry point supports nesting multiple launchers in a single script. This can be useful for tasks
that need to be run before the training code starts, such as profiling tools (dlprof), custom memory
management tools (numactl), or data preprocessing.

Example:

.. code:: bash

   dlprof --mode=simple python3 -m determined.launch.autohorovod --trial model_def:MnistTrial

**********************
 Create an Experiment
**********************

The CLI is the recommended way to create an experiment, although you can also use the WebUI to
create from an existing experiment or trial. To create an experiment:

.. code::

   $ det experiment create <configuration file> <context directory>

-  The :ref:`Experiment Configuration <experiment-configuration>` file is a YAML file that controls
   your experiment.
-  The context directory contains relevant training code, which is uploaded to the master.

The total size of the files in the context cannot exceed 95 MB. As a result, only very small
datasets should be included. Instead, set up data loaders to read data from an external source.
Refer to the :ref:`Prepare Data <prepare-data>` section for more data loading options.

Because project directories can include large artifacts that should not be packaged as part of the
model definition, including data sets or compiled binaries, users can specify a ``.detignore`` file
at the top level, which lists the file paths to be omitted from the model definition. The
``.detignore`` file uses the same syntax as `.gitignore <https://git-scm.com/docs/gitignore>`__.
Byte-compiled Python files, including ``.pyc`` files and ``__pycache__`` directories, are always
ignored.

********************
 Pre-training Setup
********************

Trials are created to train the model. The :ref:`Hyperparameter Tuning <hyperparameter-tuning>`
searcher specified in the experiment configuration file defines a set of hyperparameter
configurations. Each hyperparameter configuration corresponds to a single trial.

After the context and experiment configuration reach the master, the experiment waits for the
scheduler to assign slots. The master handles allocating necessary resources as defined in the
cluster configuration.

When a trial is ready to run, the master communicates with the agent, or :ref:`distributed training
<multi-gpu-training>` agents, which create(s) containers that have the configured environment and
training code. A set of default container images applicable to many deep learning tasks is provided,
but you can also specify a :ref:`custom image <custom-docker-images>`. If the specified container
images do not exist locally, the trial container fetches the images from the registry. See
:doc:`/post-training/model-registry`.

After starting the containers, each trial runs the ``startup-hook.sh`` script in the context
directory.

The pre-training activity can incur a delay before each trial begins training but typically only
takes a few seconds.

********************
 Pause and Activate
********************

A trial can be paused and reactivated without losing training progress. Pausing a trial preserves
its progress by saving a checkpoint before exiting the cluster.

The scheduler can pause a trial to free its resources for another task. Also, you can manually pause
an experiment, which pauses all trials in the experiment. This frees the slots used by the trial.
When the trial resumes, because more slots become available or because you activate an experiment,
the saved checkpoint is loaded and training continues from the saved state.

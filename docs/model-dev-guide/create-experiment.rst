.. _experiments:

#################################
 Create and Submit an Experiment
#################################

Run training code by submitting your code to a cluster and running it as an experiment.

************************
 Configuring a Launcher
************************

To run an experiment, specify a *launcher* in the experiment configuration's ``endpoint`` field.
Launcher options include:

-  legacy bare-Trial-class

-  Determined predefined launchers:

   -  `Horovod Launcher`_
   -  `PyTorch Distributed Launcher`_
   -  `DeepSpeed Launcher`_

-  Custom launcher

-  A command with arguments, run in the container

If you're using AMD ROCm GPUs, make sure to specify ``slot_type: rocm`` in your experiment
configuration. For more information on AMD ROCm support, see :ref:`AMD ROCm Support <rocm-support>`.

For distributed training, separate the launcher that starts distributed workers from your training
script, which typically runs each worker. The distributed training launcher must:

-  Launch all workers, passing any required peer info (e.g., rank, chief IP address) to each worker.
-  Monitor workers. If a worker exits with a non-zero status, the launcher should terminate the
   remaining workers and exit with a non-zero status.
-  Exit with zero after all workers exit with zero.

These requirements prevent distributed training jobs from hanging after a single worker failure.

*************************
 Setting Up the Launcher
*************************

The entry point of a model is configured in the :ref:`Experiment Configuration
<experiment-configuration>` file, specifying the model's path and how it should be launched.
Predefined launchers are provided in the ``determined.launch`` module, and custom scripts are
supported.

The launcher is configurable in the experiment configuration ``entrypoint`` field. The
``entrypoint`` trial object or script launches the training code.

Example entry points:

.. code:: yaml

   entrypoint: python3 -m (LAUNCHER) (TRIAL_DEFINITION)

A custom script:

.. code:: yaml

   entrypoint: python3 script.py arg1 arg2

Preconfigured launcher ``entrypoint`` arguments can differ but have the same format:

.. code:: bash

   python3 -m (LAUNCH_MODULE) (--trial TRIAL)|(SCRIPT...)

where ``(LAUNCH_MODULE)`` is a Determined launcher and ``(--trial TRIAL)|(SCRIPT...)`` refers to the
training script, which can be in a format recognized by the Determined launcher or a custom script.

For a custom launcher, wrap each rank work in the ``python3 -m determined.launch.wrap_rank $RANK CMD
[ARGS...]`` script to separate final logs by rank in the WebUI.

Training Code Definition
========================

To launch a model or training script, either pass a trial class path to ``--trial`` or run a custom
script that runs the training code. Only one of these can be used at the same time.

Trial Class
-----------

Use the ``--trial TRIAL`` argument to specify a trial class:

.. code:: bash

   --trial TRIAL

To specify a trial class to be trained, the launcher accepts a TRIAL argument in the following
format:

.. code:: yaml

   filepath:ClassName

where ``filepath`` is the location of your training class file, and ``ClassName`` is the name of the
Python training class.

Custom Script
-------------

Launch a custom script under a supported launcher with arguments as needed.

Example Python script command:

.. code:: bash

   script.py [args...]

.. _predefined-launchers:

**********************
 Predefined Launchers
**********************

Horovod Launcher
================

Format:

``determined.launch.horovod [[HVD_OVERRIDES...] --] (--trial TRIAL)|(SCRIPT...)``

The horovod launcher wraps `horovodrun
<https://horovod.readthedocs.io/en/stable/summary_include.html#running-horovod>`_ automatically
configuring workers for the trial. Pass arguments to ``horovodrun`` as ``HVD_OVERRIDES``, ending
with ``--`` to separate the overrides from the normal arguments.

Example:

.. code:: bash

   python3 -m determined.launch.horovod --fusion-threshold-mb 1 --cycle-time-ms 2 -- --trial model_def:MyTrial

.. _pytorch-dist-launcher:

PyTorch Distributed Launcher
============================

Format:

``determined.launch.torch_distributed [[TORCH_OVERRIDES...] --] (--trial TRIAL)|(SCRIPT...)``

This launcher wraps PyTorch’s native distributed training launcher, torch.distributed.run. Pass
override arguments to ``torch.distributed.run``, ending with ``--`` before the trial specification.

Example:

.. code:: bash

   python3 -m determined.launch.torch_distributed --rdzv_endpoint=$CUSTOM_RDZV_ADDR -- --trial model_def:MyTrial

DeepSpeed Launcher
==================

Format:

``determined.launch.deepspeed [[DEEPSPEED_ARGS...] --] (--trial TRIAL)|(SCRIPT...)``

The DeepSpeed launcher runs a training script under ``deepspeed``, handling IP addresses, ``sshd``
containers, and shutdown.

Example:

.. code:: bash

   python3 -m determined.launch.deepspeed --trial model_def:MyTrial

Use the ``-h`` option to get the latest usage:

.. code:: bash

   python3 -m determined.launch.deepspeed -h

.. _launch-tensorflow:

TensorFlow Launcher
===================

Format:

``determined.launch.tensorflow [--] SCRIPT...``

This launcher configures a ``TF_CONFIG`` environment variable suitable for whichever level of
TensorFlow distributed training is appropriate for the available training resources
(``MultiWorkerMirroredStrategy``, ``MirroredStrategy``, or the default strategy).

Example:

.. code:: bash

   python3 -m determined.launch.tensorflow -- python3 ./my_train.py --my-arg=value

Use the ``-h`` option to get the latest usage:

.. code:: bash

   python3 -m determined.launch.tensorflow -h

Legacy Launcher
===============

Format:

``entrypoint: model_def:TrialClass``

The entry point field supports legacy file and trial class definitions. When specifying a trial
class, it must be a subclass of a Determined trial class.

Each trial class supports one deep learning application framework. Training or validating models may
require loading data from an external source, so the training code needs to define data loaders.

A ``TrialClass`` is located in the ``model_def`` filepath and launched automatically. This legacy
configuration detects distributed training based on slot size and the number of machines, launching
with Horovod for distributed training by default. In a distributed training context, the entry point
is:

.. code:: bash

   python3 -m determined.launch.torch_distributed --trial model_def:TrialClass

Nested Launchers
================

The entry point supports nesting multiple launchers in a single script. This can be useful for tasks
that need to be run before the training code starts, such as profiling tools (dlprof), custom memory
management tools (numactl), or data preprocessing.

Example:

.. code:: bash

   dlprof --mode=simple python3 -m determined.launch.torch_distributed --trial model_def:MnistTrial

.. _creating-an-experiment:

************************
 Creating an Experiment
************************

The CLI is the recommended way to create an experiment, but you can also use the WebUI to create
from an existing experiment or trial. To create an experiment:

.. code::

   $ det experiment create <configuration file> <context directory>

-  The :ref:`Experiment Configuration <experiment-configuration>` file is a YAML file that controls
   your experiment.
-  The context directory contains relevant training code, which is uploaded to the master.

The total size of the files in the context cannot exceed 95 MB. Use data loaders to read data from
an external source for larger datasets. See the :ref:`Prepare Data <prepare-data>` section for data
loading options.

Use a ``.detignore`` file at the top level to list file paths to be omitted from the model
definition, using the same syntax as ``.gitignore``. Byte-compiled Python files, including ``.pyc``
files and ``__pycache__`` directories, are always ignored.

********************
 Pre-Training Setup
********************

Trials are created to train the model. The :ref:`Hyperparameter Tuning <hyperparameter-tuning>`
searcher specified in the experiment configuration file defines a set of hyperparameter
configurations, each corresponding to a single trial.

After the context and experiment configuration reach the master, the experiment waits for the
scheduler to assign slots. The master allocates necessary resources as defined in the cluster
configuration.

When a trial is ready to run, the master communicates with the agent or distributed training agents,
which create containers with the configured environment and training code. Default container images
applicable to many deep learning tasks are provided, but you can specify a :ref:`custom image
<custom-docker-images>`. If the specified container images do not exist locally, the trial container
fetches the images from the registry. See :ref:`organizing-models`.

.. include:: ../_shared/note-dtrain-learn-more.txt

After starting the containers, each trial runs the ``startup-hook.sh`` script in the context
directory. Pre-training activities may incur a delay before training begins, but typically only take
a few seconds.

********************
 Pause and Activate
********************

A trial can be paused and reactivated without losing training progress. Pausing a trial saves its
progress by creating a checkpoint before exiting the cluster.

The scheduler can pause a trial to free its resources for another task. You can also manually pause
an experiment, which pauses all trials in the experiment, freeing the slots used by the trial. When
the trial resumes, either because more slots become available or because you activate an experiment,
the saved checkpoint is loaded and training continues from the saved state.

See also: :ref:`Manage the job queue <job-queue>`.

.. _cifar10_pytorch_inference: https://github.com/determined-ai/determined/blob/master/examples/computer_vision/cifar10_pytorch_inference/

.. _multi-gpu-training:

######################
 Distributed Training
######################

Determined provides three main methods to take advantage of multiple GPUs:

#. **Parallelism across experiments.** Schedule multiple experiments at once: more than one
   experiment can proceed in parallel if there are enough GPUs available.

#. **Parallelism within an experiment.** Schedule multiple trials of an experiment at once: a
   :ref:`hyperparameter search <hyperparameter-tuning>` may train more than one trial at once, each
   of which will use its own GPUs.

#. **Parallelism within a trial.** Use multiple GPUs to speed up the training of a single trial
   (*distributed training*). Determined can coordinate across multiple GPUs on a single machine or
   across multiple GPUs on multiple machines to improve the performance of training a single trial.

.. note::

   In Determined, we call all of the above three methods distributed training. This might differ
   from the terminology used in other systems.

This document focuses on the third approach, demonstrating how to perform optimized distributed
training with Determined to speed up the training of a single trial.

***************
 Configuration
***************

Setting Slots Per Trial
=======================

In the :ref:`experiment-configuration`, the ``resources.slots_per_trial`` field controls the number
of GPUs that will be used to train a single trial.

The default value is 1, which disables distributed training. Setting ``slots_per_trial`` to a larger
value enables multi-GPU training automatically. Note that these GPUs might be on a single machine or
across multiple machines; the experiment configuration simply defines how many GPUs should be used
for training, and the Determined job scheduler decides whether to schedule the task on a single
agent or multiple agents, depending on the machines in the cluster and the other active workloads.

Multi-machine parallelism offers the ability to further parallelize training across more GPUs. In
order to use multi-machine parallelism, set ``slots_per_trial`` to be a multiple of the total number
of GPUs on an agent machine. For example, if your resource pool consists of 8-GPU agent machines,
valid values for M would be 16, 24, 32, etc. In this configuration, trials will use all the
resources of multiple machines to train a model.

Example configuration with distributed training:

.. code:: yaml

   resources:
     slots_per_trial: N

.. warning::

   For distributed multi-machine training, Determined automatically detects a common network
   interface shared by the agent machines. If your cluster has multiple common network interfaces,
   please specify the fastest one in :ref:`cluster-configuration` under
   ``task_container_defaults.dtrain_network_interface``.

.. note::

   When the ``slots_per_trial`` option is changed, the per-slot batch size is set to
   ``global_batch_size // slots_per_trial``. The per-slot (per-GPU) and global batch size should be
   accessed via the context using :func:`context.get_per_slot_batch_size()
   <determined.TrialContext.get_per_slot_batch_size>` and :func:`context.get_global_batch_size()
   <determined.TrialContext.get_global_batch_size>`, respectively. If ``global_batch_size`` is not
   evenly divisible by ``slots_per_trial``, the remainder is dropped.

Setting Global Batch Size
=========================

When doing distributed training, the ``global_batch_size`` specified in the
:ref:`experiment-configuration` is partitioned across ``slots_per_trial`` GPUs. The per-GPU batch
size is set to: ``global_batch_size`` / ``slots_per_trial``. If ``slots_per_trial`` does not divide
the ``global_batch_size`` evenly, the batch size is rounded down. For convenience, the per-GPU batch
size can be accessed via the Trial API, using :func:`context.get_per_slot_batch_size
<determined.TrialContext.get_per_slot_batch_size>`.

For improved performance, we recommend *weak-scaling*: increasing your ``global_batch_size``
proportionally with ``slots_per_trial`` (e.g., change ``global_batch_size`` of 32 for
``slots_per_trial`` of 1 to ``global_batch_size`` of 128 for ``slots_per_trial`` of 4).

Adjusting ``global_batch_size`` can affect your model convergence, which can affect your training
and/or testing accuracy. You may need to adjust model hyperparameters like the learning rate and/or
use a different optimizer when training with larger batch sizes.

Advanced Optimizations
======================

Determined supports several optimizations to further reduce training time. These optimizations are
available in :ref:`experiment-configuration` under ``optimizations``.

-  ``optimizations.aggregation_frequency`` controls how many batches are evaluated before exchanging
   gradients. It is helpful in situations where it is not possible to increase the batch size
   directly (e.g., due to GPU memory limitations). This optimization increases your effective batch
   size to ``aggregation_frequency`` * ``global_batch_size``.

-  ``optimizations.gradient_compression`` reduces the time it takes to transfer gradients between
   GPUs.

-  ``optimizations.auto_tune_tensor_fusion`` automatically identifies the optimal message size
   during gradient transfers, reducing communication overhead.

-  ``optimizations.average_training_metrics`` averages the training metrics across GPUs at the end
   of every training workload, which requires communication. This will typically not have a major
   impact on training performance, but if you have a very small ``scheduling_unit``, ensuring it is
   disabled may improve performance. If this option is disabled (which is the default behavior),
   only the training metrics from the chief GPU are used. This impacts shown in the Determined UI
   and TensorBoard, but does not influence model behavior or hyperparameter search.

If you do not see improved performance using distributed training, there might be a performance
bottleneck in the model that cannot be directly alleviated by using multiple GPUs, e.g., data
loading. We suggest experimenting with a synthetic dataset to verify the performance of multi-GPU
training.

.. warning::

   Multi-machine distributed training is designed to maximize performance by training with all the
   resources of a machine. This can lead to situations where an experiment is created but never
   becomes active: if the number of GPUs requested does not divide into the machines available, for
   instance, or if another experiment is already using some GPUs on a machine.

   If an experiment does not become active after a minute or so, please confirm that
   ``slots_per_trial`` is a multiple of the number of GPUs available on a machine. You can also use
   the CLI command ``det task list`` to check if any other tasks are using GPUs and preventing your
   experiment from using all the GPUs on a machine.

******************
 Data Downloading
******************

When performing distributed training, Determined will automatically create one process for every GPU
that is being used for training. Each process will attempt to download training and/or validation
data, so care should be taken to ensure that concurrent data downloads do not conflict with one
another. One way to do this is to include a unique identifier in the local file system path where
the downloaded data is stored. A convenient identifier is the ``rank`` of the current process: a
process's ``rank`` is automatically assigned by Determined, and will be unique among all the
processes in a trial.

You can do this by leveraging the :func:`self.context.distributed.get_rank()
<determined.core._distributed.DistributedContext.get_rank>` function. Below is an example of how to
do this when downloading data from S3. In this example, the S3 bucket name is configured via a field
``data.bucket`` in the experiment configuration.

.. code:: python

   import boto3
   import os


   def download_data_from_s3(self):
       s3_bucket = self.context.get_data_config()["bucket"]
       download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
       data_file = "data.csv"

       s3 = boto3.client("s3")
       os.makedirs(download_directory, exist_ok=True)
       filepath = os.path.join(download_directory, data_file)
       if not os.path.exists(filepath):
           s3.download_file(s3_bucket, data_file, filepath)
       return download_directory

.. _dtrain-scheduling:

*********************
 Scheduling Behavior
*********************

The Determined master takes care of scheduling distributed training jobs automatically, ensuring
that all of the compute resources required for a job are available before the job itself is
launched. Users should be aware of the following details about scheduler behavior when using
distributed training:

-  If ``slots_per_trial`` is smaller than or equal to the number of slots on a single agent,
   Determined will consider scheduling multiple distributed training jobs on a single agent. This is
   designed to improve utilization and to allow multiple small training jobs to run on a single
   agent. For example, an agent with 8 GPUs could be assigned two 4-GPU jobs, or four 2-GPU jobs.

-  Otherwise, if ``slots_per_trial`` is greater than the number of slots on a single agent,
   Determined will schedule the distributed training job onto multiple agents. A multi-machine
   distributed training job will only be scheduled onto an agent if this will result in utilizing
   all of the agent's GPUs. This is to ensure good performance and utilize the full network
   bandwidth of each machine, while minimizing inter-machine networking. For example, if all of the
   agents in your cluster have 8 GPUs each , you should submit jobs with ``slots_per_trial`` set to
   a multiple of 8 (e.g., 8, 16, or 24).

.. warning::

   If the scheduling constraints for multi-machine distributed training described above are not
   satisfied, distributed training jobs will not be scheduled and will wait indefinitely. For
   example, if every agent in the cluster has 8 GPUs, a job with ``slots_per_trial`` set to ``12``
   will never be scheduled.

   If a multi-GPU experiment does not become active after a minute or so, please confirm that
   ``slots_per_trial`` is set so that it can be scheduled within these constraints. The CLI command
   ``det task list`` can also be used to check if any other tasks are using GPUs and preventing your
   experiment from using all the GPUs on a machine.

***********************
 Distributed Inference
***********************

PyTorch users can also use the existing distributed training workflow with PyTorchTrial to
accelerate their inference workloads. This workflow is not yet officially supported, so users must
specify certain training-specific artifacts that are not used for inference. To run a distributed
batch inference job, create a new PyTorchTrial and follow these steps:

-  Load the trained model and build the inference dataset using ``build_validation_data_loader()``.
-  Specify the inference step using ``evaluate_batch()`` or ``evaluate_full_dataset()``.
-  Register a dummy ``optimizer``.
-  Specify a ``build_training_data_loader()`` that returns a dummy dataloader.
-  Specify a no-op ``train_batch()`` that returns an empty map of metrics.

Once the new PyTorchTrial object is created, use the experiment configuration to distribute
inference in the same way as training. cifar10_pytorch_inference_ is an example of distributed batch
inference.

*****
 FAQ
*****

Why do my distributed training experiments never start?
=======================================================

If :ref:`slots_per_trial <exp-config-resources-slots-per-trial>` is greater than the number of slots
on a single agent, Determined will schedule it over multiple machines. When scheduling a
multi-machine distributed training job, Determined requires that the job uses all of the slots
(GPUs) on an agent. For example, in a cluster that consists of 8-GPU agents, an experiment with
:ref:`slots_per_trial <exp-config-resources-slots-per-trial>` set to ``12`` will never be scheduled
and will instead wait indefinitely. The :ref:`distributed training documentation
<dtrain-scheduling>` describes this scheduling behavior in more detail.

There may also be running tasks preventing your multi-GPU trials from acquiring enough GPUs on a
single machine. Consider adjusting ``slots_per_trial`` or terminating existing tasks to free up
slots in your cluster.

Why do my multi-machine training experiments appear to be stuck?
================================================================

Multi-machine training requires that all machines are able to connect to each other directly. There
may be firewall rules or network configuration that prevent machines in your cluster from
communicating. Please check if agent machines can access each other outside of Determined (e.g.,
using the ``ping`` or ``netcat`` tools).

More rarely, if agents have multiple network interfaces and some of them are not routable,
Determined may pick one of those interfaces rather than one that allows one agent to contact
another. In this case, it is possible to set the network interface used for distributed training
explicitly in the :ref:`cluster-configuration`.

.. toctree::
   :maxdepth: 1
   :hidden:

   effective-distributed-training

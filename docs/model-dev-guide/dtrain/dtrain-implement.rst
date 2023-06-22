.. _multi-gpu-training-implement:

###################################
 Implementing Distributed Training
###################################

**************
 Connectivity
**************

Multi-machine training necessitates that all machines are capable of establishing a direct
connection. Firewall rules or network configurations might exist that prevent machines in your
cluster from communicating with each other. You can verify that agent machines can connect with each
other outside of Determined by using tools such as ``ping`` or ``netcat``.

More rarely, if agents have multiple network interfaces and some of them are not routable,
Determined may pick one of those interfaces rather than one that allows one agent to contact
another. In this case, it is possible to explicitly set the network interface used for distributed
training, as described in :ref:`Basic Setup: Step 7 - Configure the Cluster
<cluster-configuration>`.

***************
 Configuration
***************

Slots Per Trial
===============

The ``resources.slots_per_trial`` field in the :ref:`experiment configuration
<experiment-config-reference>` controls the number of GPUs used to train a single trial.

By default, this field is set to a value of ``1``, which disables distributed training. If you
increase the ``slots_per_trial`` value, this will automatically enable multi-GPU training. Bear in
mind that these GPUs can either be located on a single machine or distributed across multiple
machines. The experiment configuration merely dictates the number of GPUs to be used in the training
process, while the Determined job scheduler decides whether to schedule the task on a single agent
or multiple agents. Whether the job scheduler schedules the task on a single agent or multiple
agents depends on the machines in the cluster and other active workloads.

Multi-machine parallelism allows you to further parallelize training across more GPUs. To use this
feature, set ``slots_per_trial`` to a multiple of the total number of GPUs on an agent machine. For
example, if your resource pool consists of multiple 8-GPU agent machines, valid ``slots_per_trial``
values would be 16, 24, 32, and so on.

In the following configuration, trials will use the combined resources of multiple machines to train
a model:

.. code:: yaml

   resources:
     slots_per_trial: 16  # Two 8-GPU agent machines will be used in a trial

For distributed multi-machine training, Determined will automatically detect a common network
interface that is shared by the agent machines. If your cluster has multiple common network
interfaces, we advise specifying the fastest one in :ref:`cluster-configuration` under
``task_container_defaults.dtrain_network_interface``.

When the ``slots_per_trial`` field is set, the per-slot (i.e., per-GPU) batch size is set to
``global_batch_size // slots_per_trial``. The per-slot and global batch sizes can be accessed
through the context using :func:`context.get_per_slot_batch_size()
<determined.TrialContext.get_per_slot_batch_size>` and :func:`context.get_global_batch_size()
<determined.TrialContext.get_global_batch_size>`, respectively. If ``global_batch_size`` is not
evenly divisible by ``slots_per_trial``, the remainder is dropped.

When scheduling a multi-machine distributed training job, Determined prefers that the job use all of
the slots (GPUs) on an agent. The section on :ref:`Scheduling Behavior <dtrain-scheduling>`
describes this preference in more detail.

.. note::

   You might have existing tasks that are running on a single machine that are preventing your
   multi-GPU trials from acquiring sufficient GPUs. To alleviate this, you may want to consider
   adjusting ``slots_per_trial`` or terminating existing tasks to free up slots in your cluster.

Global Batch Size
=================

You can reduce computational overhead by setting the ``global_batch_size`` to the largest batch size
that fits into a single GPU multiplied times the number of slots.

.. note::

   This feature only applies to :ref:`high-level-apis` (Trial APIs) and does not apply to the Core
   API.

During distributed training, the ``global_batch_size`` specified in the :ref:`experiment
configuration file <experiment-config-reference>` is partitioned across ``slots_per_trial`` GPUs.
The per-GPU batch size is set to: ``global_batch_size // slots_per_trial``. Recall that if
``global_batch_size`` is not evenly divisible by ``slots_per_trial``, the remainder is dropped. For
convenience, the per-GPU batch size can be accessed via the Trial API, using
:func:`context.get_per_slot_batch_size <determined.TrialContext.get_per_slot_batch_size>`.

For improved performance, *weak-scaling* is recommended. Weak-scaling means proportionally
increasing your ``global_batch_size`` with ``slots_per_trial``. For example, you might change
``global_batch_size`` and ``slots_per_trial`` from 32 and 1 to 128 and 4, respectively. You can
visit the blog post, `Scaling deep learning workloads
<https://developer.hpe.com/blog/scaling-deep-learning-workloads/>`_, to learn more about weak
scaling.

Note that adjusting ``global_batch_size`` can impact your model convergence, which in turn can
affect your training and/or testing accuracy. You might need to adjust model hyperparameters, such
as the learning rate, or consider using a different optimizer when training with larger batch sizes.

.. _multi-gpu-training-implement-adv-optimizations:

Advanced Optimizations
======================

The following optimizations can further reduce training time.

-  ``optimizations.aggregation_frequency`` controls how many batches are evaluated before exchanging
   gradients. This optimization increases your effective batch size to ``aggregation_frequency`` *
   ``global_batch_size``. ``optimizations.aggregation_frequency`` is useful in scenarios where
   directly increasing the batch size is not possible (for example, due to GPU memory limitations).

-  ``optimizations.gradient_compression`` reduces the time it takes to transfer gradients between
   GPUs.

-  ``optimizations.auto_tune_tensor_fusion`` automatically identifies the optimal message size
   during gradient transfers, thereby reducing communication overhead.

-  ``optimizations.average_training_metrics`` averages the training metrics across GPUs at the end
   of every training workload, a process that requires communication. ``average_training_metrics``
   is set to ``true`` by default and typically does not have a significant impact on training
   performance. However, if you have a very small ``scheduling_unit``, disabling this option could
   improve performance. When disabled, only the training metrics from the chief GPU are reported.
   This impacts results shown in the WebUI and TensorBoard but does not influence model behavior or
   hyperparameter search.

To learn more about these optimizations, visit the :ref:`optimizations <exp-config-optimizations>`
section in the Experiment Configuration Reference.

If you're not seeing improved performance with distributed training, your model might have a
performance bottleneck that can't be directly alleviated by using multiple GPUs, such as with data
loading. You're encouraged to experiment with a synthetic dataset in order to verify the performance
of multi-GPU training.

.. warning::

   Multi-machine distributed training is designed to maximize performance by training with all the
   resources of a machine. This can lead to situations where an experiment is created but never
   becomes active, such as when the number of GPUs requested does not factor into (divide evenly)
   the machines available, or when another experiment is already using some GPUs on a machine.

   If an experiment does not become active after a minute or so, please ensure that
   ``slots_per_trial`` is a multiple of the number of GPUs available on a machine. You can also use
   the CLI command ``det task list`` to check if any other tasks are using GPUs and preventing your
   experiment from using all the GPUs on a machine.

******************
 Downloading Data
******************

When performing distributed training, Determined automatically creates one process for each GPU that
is being used for training. Each of these processes attempts to download training and/or validation
data, so it is important to ensure that concurrent data downloads do not conflict with one another.

One way to achieve this is to include a unique identifier in the local file system path where the
downloaded data is stored. A convenient identifier is the ``rank`` of the current process. The
process ``rank`` is automatically assigned by Determined and is unique among all trial processes.
You can accomplish this by leveraging the :func:`self.context.distributed.get_rank()
<determined._core._distributed.DistributedContext.get_rank>` function.

The following example demonstrates how to accomplish this when downloading data from S3. In this
example, the S3 bucket name is configured via a ``data.bucket`` field in the experiment
configuration file.

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

The Determined master schedules distributed training jobs automatically, ensuring that all of the
compute resources required for a job are available before the job is launched. Here are some
important details regarding ``slots_per_trial`` and the scheduler's behavior:

-  If ``slots_per_trial`` is less than or equal to the number of slots on a single agent, Determined
   considers scheduling multiple distributed training jobs on a single agent. This approach is
   designed to improve utilization and to allow multiple small training jobs to run on a single
   agent. For example, an agent with eight GPUs could be assigned two 4-GPU jobs or four 2-GPU jobs.

-  If ``slots_per_trial`` is greater than the number of slots on a single agent, Determined
   schedules the distributed training job onto multiple agents. To ensure good performance and
   utilize the full network bandwidth of each machine and to minimize inter-machine networking,
   Determined prefers utilizing all of the agent GPUs on a machine. For example, if all the agents
   in your cluster have eight GPUs each, you should submit jobs with ``slots_per_trial`` set to a
   multiple of eight, such as 8, 16, or 24.

.. note::

   The scheduler can find fits for distributed jobs against agents of different sizes. This is
   configured via the :ref:`allowing_heterogeneous_fits <allow-uneven-slots>` parameter. This
   parameter defaults to ``false``. By default Determined requires that the job use all of the slots
   (GPUs) on an agent.

.. warning::

   If these scheduling constraints for multi-machine distributed training are not satisfied, and you
   have not configured the :ref:`allowing_heterogeneous_fits <allow-uneven-slots>` parameter,
   distributed training jobs are not scheduled and wait indefinitely. For example, if every agent in
   the cluster has eight GPUs, a job with ``slots_per_trial`` set to ``12`` is never scheduled.

   If a multi-GPU experiment does not become active after a minute or so, please ensure that
   ``slots_per_trial`` is set so that it can be scheduled within these constraints. You can also use
   the CLI command ``det task list`` to check if any other tasks are using GPUs and preventing your
   experiment from using all the GPUs on a machine.

***********************
 Distributed Inference
***********************

PyTorch users have the option to use the existing distributed training workflow with PyTorchTrial to
accelerate their inference workloads. This workflow is not yet officially supported, therefore,
users must specify certain training-specific artifacts that are not used for inference. To run a
distributed batch inference job, create a new PyTorchTrial and follow these steps:

-  Load the trained model and build the inference dataset using ``build_validation_data_loader()``.
-  Specify the inference step using ``evaluate_batch()`` or ``evaluate_full_dataset()``.
-  Register a dummy ``optimizer``.
-  Specify a ``build_training_data_loader()`` that returns a dummy dataloader.
-  Specify a no-op ``train_batch()`` that returns an empty map of metrics.

Once the new PyTorchTrial object is created, use the experiment configuration to distribute
inference in the same way as training. `cifar10_pytorch_inference
<https://github.com/determined-ai/determined/blob/master/examples/computer_vision/cifar10_pytorch_inference/>`_
serves as an example of distributed batch inference.

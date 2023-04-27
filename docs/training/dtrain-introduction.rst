.. _multi-gpu-training:

######################################
 Distributed Training with Determined
######################################

This overview demonstrates how to perform optimized distributed training with Determined to speed up
the training of a single trial. The first section conceptually describes distributed training and is
followed by a section on implementing distributed training.

******************
 Concept Overview
******************

How Determined Distributed Training Works
=========================================

Distributed training in Determined utilizes data-parallelism. Data-parallelism for deep-learning
consists of a set of workers, where each worker is assigned to a unique compute accelerator such as
a GPU or a TPU. Each worker maintains a copy of the model parameters (weights that are being
trained), which is synchronized across all the workers at the start of training.

After initialization is completed, distributed training in Determined follows a loop where:

#. Every worker performs a forward and backward pass on a unique mini-batch of data.
#. As the result of the backward pass, every worker generates a set of updates to the model
   parameters based on the data it processed.
#. The workers communicate their updates to each other, so that all the workers see all the updates
   made during that batch.
#. Every worker averages the updates by the number of workers.
#. Every worker applies the updates to its copy of the model parameters, resulting in all the
   workers having identical solution states.
#. Return to the first step.

Reduce Computation and Communication Overhead
=============================================

Of the steps involved in the distributed training loop in Determined, which are described above,
step 1 and step 2 introduce the majority of the computational overhead. To reduce computational
overhead, it's recommended that users maximize the utilization of their GPU. This is typically done
by using the largest batch size that fits into memory. When performing distributed training, to
reduce the computational overhead it's recommended to set the ``global_batch_size`` to the ``largest
batch size that fits into a single GPU`` * ``number of slots``. This is commonly referred to as
*weak scaling*.

Step 3 of the distributed training loop in Determined introduces the majority of the communication
overhead. Because deep learning models typically perform dense updates, where every model parameter
is updated for every training sample, ``batch_size`` does not affect how long it takes workers to
communicate updates. However, increasing ``global_batch_size`` does reduce the required number of
passes through the training loop, thus reducing the total communication overhead.

Determined optimizes the communication in step 3 by using an efficient form of ring all-reduce,
which minimizes the amount of communication necessary for all the workers to communicate their
updates. Determined also reduces the communication overhead by overlapping computation (step 1 &
step 2) and communication (step 3) by communicating updates for deeper layers concurrently with
computing updates for the shallower layers. The :ref:`multi-gpu-training` document covers additional
optimizations available in Determined for reducing the communication overhead.

How to Train Effectively with Large Batch Sizes
===============================================

To improve the performance of distributed training, we recommend using the largest possible
``global_batch_size``, setting it to be ``largest batch size that fits into a single GPU`` *
``number of slots``. However, training with a large ``global_batch_size`` can have adverse effects
on the convergence (accuracy) of the model. At Determined AI we have found several effective
techniques for training with large batch sizes:

-  Starting with the ``original learning rate`` used for a single GPU and gradually increasing it to
   ``number of slots`` * ``original learning rate`` throughout the first several epochs. For more
   details, see `Accurate, Large Minibatch SGD: Training ImageNet in 1 Hour
   <https://arxiv.org/pdf/1706.02677.pdf>`_.

-  Using custom optimizers designed for large batch training, such as `RAdam
   <https://github.com/LiyuanLucasLiu/RAdam>`_, `LARS <https://arxiv.org/pdf/1708.03888.pdf>`_, or
   `LAMB <https://arxiv.org/pdf/1904.00962.pdf>`_. We have found RAdam especially effective.

These techniques often require hyperparameter modifications. To automate this process, we encourage
users to utilize the :ref:`hyperparameter-tuning` capabilities in Determined.

Model Characteristics that Affect Performance
=============================================

Deep learning models typically perform dense updates, where every model parameter is updated for
every training sample. Because of this, the amount of communication per mini-batch (step 3 in the
distributed training loop) is dependent on the number of model parameters. Models that have fewer
parameters such as `ResNet-50 <https://arxiv.org/pdf/1512.03385.pdf>`_ (~30 million parameters)
train more efficiently in distributed settings than models with more parameters such as `VGG-16
<https://arxiv.org/pdf/1505.06798.pdf>`_ (~136 million parameters). If planning to utilize
distributed training, we encourage users to be mindful of their model size when designing models.

Debug Performance Bottlenecks
=============================

When scaling up distributed training, it's fairly common to see non-linear speedup when scaling from
one machine to two machines as intra-machine communication (e.g., NVLink) is often significantly
faster than inter-machine communication. Scaling up beyond two machines often provides close to
linear speed-up, but it does vary depending on the model characteristics. If observing unexpected
scaling performance, assuming you have scaled your ``global_batch_size`` proportionally with
``slots_per_trial``, it's possible that training performance is being bottlenecked by network
communication or disk I/O.

To check if your training is bottlenecked by communication, we suggest setting
``optimizations.aggregation_frequency`` in the :ref:`experiment-config-reference` to a very large
number (e.g., 1000). This setting results in communicating updates once every 1000 batches.
Comparing throughput with ``aggregation_frequency`` of 1 vs. ``aggregation_frequency`` of 1000 will
demonstrate the communication overhead. If you do observe significant communication overhead, refer
to :ref:`multi-gpu-training` for guidance on how to optimize communication.

To check if your training is bottlenecked by I/O, we encourage users to experiment with using
synthetic datasets. If you observe that I/O is a significant bottleneck, we suggest optimizing the
data input pipeline to the model (e.g., copy training data to local SSDs).

.. _reproducibility:

Reproducibility
===============

Determined aims to support *reproducible* machine learning experiments: that is, the result of
running a Determined experiment should be deterministic, so that rerunning a previous experiment
should produce an identical model. For example, this ensures that if the model produced from an
experiment is ever lost, it can be recovered by rerunning the experiment that produced it.

Status
------

The current version of Determined provides limited support for reproducibility; unfortunately, the
current state of the hardware and software stack typically used for deep learning makes perfect
reproducibility very challenging.

Determined can control and reproduce the following sources of randomness:

-  Hyperparameter sampling decisions.
-  The initial weights for a given hyperparameter configuration.
-  Shuffling of training data in a trial.
-  Dropout or other random layers.

Determined currently does not offer support for controlling non-determinism in floating-point
operations. Modern deep learning frameworks typically implement training using floating point
operations that result in non-deterministic results, particularly on GPUs. If only CPUs are used for
training, reproducible results can be achieved, as described in the following sections.

Random Seeds
------------

Each Determined experiment is associated with an **experiment seed**: an integer ranging from 0 to
2\ :sup:`31`--1. The experiment seed can be set using the ``reproducibility.experiment_seed`` field
of the experiment configuration. If an experiment seed is not explicitly specified, the master will
assign one automatically.

The experiment seed is used as a source of randomness for any hyperparameter sampling procedures.
The experiment seed is also used to generate a **trial seed** for every trial associated with the
experiment.

In the ``Trial`` interface, the trial seed is accessible within the trial class using
``self.ctx.get_trial_seed()``.

Coding Guidelines
-----------------

To achieve reproducible initial conditions in an experiment, please follow these guidelines:

-  Use the `np.random <https://docs.scipy.org/doc/numpy-1.14.0/reference/routines.random.html>`__ or
   `random <https://docs.python.org/3/library/random.html>`__ APIs for random procedures, such as
   shuffling of data. Both PRNGs will be initialized with the trial seed by Determined
   automatically.

-  Use the trial seed to seed any randomized operations (e.g., initializers, dropout) in your
   framework of choice. For example, Keras `initializers <https://keras.io/initializers/>`__ accept
   an optional seed parameter. Again, it is not necessary to set any *graph-level* PRNGs (e.g.,
   TensorFlow's ``tf.set_random_seed``), as Determined manages this for you.

Deterministic Floating Point on CPUs
------------------------------------

When doing CPU-only training with TensorFlow, it is possible to achieve floating-point
reproducibility throughout optimization. If using the :class:`~determined.keras.TFKerasTrial` API,
implement the optional :meth:`~determined.keras.TFKerasTrial.session_config` method to override the
default session configuration:

.. code:: python

   def session_config(self) -> tf.ConfigProto:
       return tf.ConfigProto(
           intra_op_parallelism_threads=1, inter_op_parallelism_threads=1
       )

.. warning::

   Disabling thread parallelism may negatively affect performance. Only enable this feature if you
   understand and accept this trade-off.

Pause Experiments
-----------------

TensorFlow does not fully support the extraction or restoration of a single, global RNG state.
Consequently, pausing experiments that use a TensorFlow-based framework may introduce an additional
source of entropy.

Optimize Training
=================

When optimizing the training speed of a model, the first step is to understand where and why
training is slow. Once the bottlenecks have been identified, the next step is to do further
investigation and experimentation to alleviate those bottlenecks.

To understand the performance profile of a training job, the training code and infrastructure need
to be instrumented. There are many different layers that can be instrumented, from raw throughput
all the way down to GPU kernels.

Determined provides two tools out-of-the-box for instrumenting training:

-  :ref:`System Metrics <how-to-profiling-system-metrics>`: measurements of hardware usage
-  :ref:`Timings <how-to-profiling-timings>`: durations of actions taken during training, such as
   dataloading

System Metrics are useful to see if the software is taking full advantage of the available hardware,
particularly around GPU usage, dataloading, and network communication during distributed training.
Timings are useful for identifying the section of code to focus on for optimizations. Most commonly,
Timings help answer the question of whether the dataloader is the main bottleneck in training.

.. _how-to-profiling:

.. _how-to-profiling-system-metrics:

System Metrics
--------------

System Metrics are statistics around hardware usage, such as GPU utilization and network throughput.
These metrics are useful for seeing whether training is using the hardware effectively. When the
System Metrics reported for an experiment are below what is expected from the hardware, that is a
sign that the software may be able to be optimized to make better use of the hardware resources.

Specifically, Determined tracks:

-  GPU utilization
-  GPU free memory
-  Network throughput (sent)
-  Network throughput (received)
-  Disk IOPS
-  Disk throughput (read)
-  Disk throughput (write)
-  Host available memory
-  CPU utilization averaged across cores

For distributed training, these metrics are collected for every agent. The data are broken down by
agent, and GPU metrics can be further broken down by GPU.

.. note::

   System Metrics record agent-level metrics, so when there are multiple experiments on the same
   agent, it is difficult to analyze. We suggest that profiling is done with only a single
   experiment per agent.

.. _how-to-profiling-timings:

Timings
-------

The other type of profiling metric that Determined tracks is Timings. Timings are measurements of
how long specific training events take. Examples of training events include retrieving data from the
dataloader, moving data between host and device, running the forward/backward pass, and executing
callbacks.

.. note::

   Timings are currently only supported for ``PyTorchTrial``.

These measurements provide a high-level picture of where to focus optimization efforts.
Specifically, Determined tracks the following Timings:

-  ``dataloader_next``: time to retrieve the next item from the dataloader
-  ``to_device``: time to transfer input from host to device
-  ``train_batch``: how long the user-defined ``train_batch`` function takes to execute\*
-  ``step_lr_schedulers``: amount time taken to update the LR schedules
-  ``from_device``: time to transfer output from device to host
-  ``reduce_metrics``: time taken to calculate global metrics in distributed training

\* ``train_batch`` is typically the forward pass and the backward pass, but it is a user-defined
function so it could include other steps.

***************************************************
 Introduction to Implementing Distributed Training
***************************************************

Connectivity
============

Multi-machine training requires that all machines can connect directly. There may be firewall rules
or network configuration that prevent machines in your cluster from communicating. Please check that
agent machines can access each other outside of Determined by using ``ping`` or ``netcat`` tools.

More rarely, if agents have multiple network interfaces and some of them are not routable,
Determined may pick one of those interfaces rather than one that allows one agent to contact
another. In this case, it is possible to explicitly set the network interface used for distributed
training, as described in :ref:`cluster-configuration`.

Configuration
=============

Slots Per Trial
---------------

In the :ref:`experiment-config-reference`, the ``resources.slots_per_trial`` field controls the
number of GPUs used to train a single trial.

The default value is ``1``, which disables distributed training. Setting ``slots_per_trial`` to a
larger value enables multi-GPU training automatically. Note that these GPUs might be on a single
machine or across multiple machines; the experiment configuration simply defines how many GPUs
should be used for training, and the Determined job scheduler decides whether to schedule the task
on a single agent or multiple agents, depending on the machines in the cluster and the other active
workloads.

Multi-machine parallelism offers the ability to further parallelize training across more GPUs. To
use multi-machine parallelism, set ``slots_per_trial`` to be a multiple of the total number of GPUs
on an agent machine. For example, if your resource pool consists of multiple 8-GPU agent machines,
valid values for ``slots_per_trial`` would be 16, 24, 32, etc. In this configuration, trials use all
the resources of multiple machines to train a model:

.. code:: yaml

   resources:
     slots_per_trial: 16  # Two 8-GPU agent machines will be used in a trial

For distributed multi-machine training, Determined automatically detects a common network interface
shared by the agent machines. If your cluster has multiple common network interfaces, please specify
the fastest one in :ref:`cluster-configuration` under
``task_container_defaults.dtrain_network_interface``.

When the ``slots_per_trial`` field is set, the per-slot (i.e., per-GPU) batch size is set to
``global_batch_size // slots_per_trial``. The per-slot and global batch sizes should be accessed via
the context using :func:`context.get_per_slot_batch_size()
<determined.TrialContext.get_per_slot_batch_size>` and :func:`context.get_global_batch_size()
<determined.TrialContext.get_global_batch_size>`, respectively. If ``global_batch_size`` is not
evenly divisible by ``slots_per_trial``, the remainder is dropped.

If :ref:`slots_per_trial <exp-config-resources-slots-per-trial>` is greater than the number of slots
on a single agent, Determined schedules it over multiple machines. When scheduling a multi-machine
distributed training job, Determined requires that the job uses all of the slots (GPUs) on an agent.
For example, in a cluster that consists of 8-GPU agents, an experiment with :ref:`slots_per_trial
<exp-config-resources-slots-per-trial>` set to ``12`` is never scheduled and will wait indefinitely.
The section on :ref:`Scheduling Behavior <dtrain-scheduling>` describes this in more detail.

There might also be running tasks preventing your multi-GPU trials from acquiring enough GPUs on a
single machine. Consider adjusting ``slots_per_trial`` or terminating existing tasks to free slots
in your cluster.

Global Batch Size
-----------------

When doing distributed training, the ``global_batch_size`` specified in the
:ref:`experiment-config-reference` is partitioned across ``slots_per_trial`` GPUs. The per-GPU batch
size is set to: ``global_batch_size // slots_per_trial``. If ``slots_per_trial`` does not divide
``global_batch_size`` evenly, the remainder is dropped. For convenience, the per-GPU batch size can
be accessed via the Trial API, using :func:`context.get_per_slot_batch_size
<determined.TrialContext.get_per_slot_batch_size>`.

For improved performance, *weak-scaling* is recommended. That is, increasing your
``global_batch_size`` proportionally with ``slots_per_trial``. For example, change
``global_batch_size`` and ``slots_per_trial`` from 32 and 1 to 128 and 4.

Adjusting ``global_batch_size`` can affect your model convergence, which can affect your training
and/or testing accuracy. You may need to adjust model hyperparameters like the learning rate and/or
use a different optimizer when training with larger batch sizes.

Advanced Optimizations
----------------------

Determined supports several optimizations to further reduce training time. These optimizations are
available in :ref:`experiment-config-reference` under ``optimizations``.

-  ``optimizations.aggregation_frequency`` controls how many batches are evaluated before exchanging
   gradients. It is helpful in situations where it is not possible to increase the batch size
   directly, for example, due to GPU memory limitations). This optimization increases your effective
   batch size to ``aggregation_frequency`` * ``global_batch_size``.

-  ``optimizations.gradient_compression`` reduces the time it takes to transfer gradients between
   GPUs.

-  ``optimizations.auto_tune_tensor_fusion`` automatically identifies the optimal message size
   during gradient transfers, reducing communication overhead.

-  ``optimizations.average_training_metrics`` averages the training metrics across GPUs at the end
   of every training workload, which requires communication. ``average_training_metrics`` is set to
   ``true`` by default. This typically does not have a major impact on training performance, but if
   you have a very small ``scheduling_unit``, disabling this option may improve performance. When
   disabled, only the training metrics from the chief GPU are reported. This impacts results shown
   in the WebUI and TensorBoard but does not influence model behavior or hyperparameter search.

If you do not see improved performance using distributed training, there might be a performance
bottleneck in the model that cannot be directly alleviated by using multiple GPUs, such as with data
loading. You are encouraged to experiment with a synthetic dataset to verify the performance of
multi-GPU training.

.. warning::

   Multi-machine distributed training is designed to maximize performance by training with all the
   resources of a machine. This can lead to situations where an experiment is created but never
   becomes active: if the number of GPUs requested does not divide into the machines available, for
   instance, or if another experiment is already using some GPUs on a machine.

   If an experiment does not become active after a minute or so, please confirm that
   ``slots_per_trial`` is a multiple of the number of GPUs available on a machine. You can also use
   the CLI command ``det task list`` to check if any other tasks are using GPUs and preventing your
   experiment from using all the GPUs on a machine.

Downloading Data
================

When performing distributed training, Determined automatically creates one process for every GPU
that is being used for training. Each process attempts to download training and/or validation data,
so care should be taken to ensure that concurrent data downloads do not conflict with one another.
One way to do this is to include a unique identifier in the local file system path where the
downloaded data is stored. A convenient identifier is the ``rank`` of the current process: the
process ``rank`` is automatically assigned by Determined and is unique among all trial processes.

You can do this by leveraging the :func:`self.context.distributed.get_rank()
<determined._core._distributed.DistributedContext.get_rank>` function. Below is an example of how to
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

Scheduling Behavior
===================

The Determined master takes care of scheduling distributed training jobs automatically, ensuring
that all of the compute resources required for a job are available before the job itself is
launched. Users should be aware of the following details about scheduler behavior when using
distributed training:

-  If ``slots_per_trial`` is smaller than or equal to the number of slots on a single agent,
   Determined considers scheduling multiple distributed training jobs on a single agent. This is
   designed to improve utilization and to allow multiple small training jobs to run on a single
   agent. For example, an agent with eight GPUs could be assigned two 4-GPU jobs or four 2-GPU jobs.

-  Otherwise, if ``slots_per_trial`` is greater than the number of slots on a single agent,
   Determined schedules the distributed training job onto multiple agents. A multi-machine
   distributed training job is only scheduled onto an agent if this results in utilizing all of the
   agent GPUs. This is to ensure good performance and utilize the full network bandwidth of each
   machine while minimizing inter-machine networking. For example, if all of the agents in your
   cluster have eight GPUs each , you should submit jobs with ``slots_per_trial`` set to a multiple
   of eight, such as 8, 16, or 24.

.. warning::

   If the scheduling constraints for multi-machine distributed training described above are not
   satisfied, distributed training jobs are not scheduled and wait indefinitely. For example, if
   every agent in the cluster has eight GPUs, a job with ``slots_per_trial`` set to ``12`` is never
   scheduled.

   If a multi-GPU experiment does not become active after a minute or so, please confirm that
   ``slots_per_trial`` is set so that it can be scheduled within these constraints. The CLI command
   ``det task list`` can also be used to check if any other tasks are using GPUs and preventing your
   experiment from using all the GPUs on a machine.

Distributed Inference
=====================

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
inference in the same way as training. `cifar10_pytorch_inference
<https://github.com/determined-ai/determined/blob/master/examples/computer_vision/cifar10_pytorch_inference/>`_
is an example of distributed batch inference.

.. _config-template:

Configuration Templates
=======================

At a typical organization, many Determined configuration files will contain similar settings. For
example, all of the training workloads run at a given organization might use the same checkpoint
storage configuration. One way to reduce this redundancy is to use *configuration templates*. With
this feature, users can move settings that are shared by many experiments into a single YAML file
that can then be referenced by configurations that require those settings.

Each configuration template has a unique name and is stored by the Determined master. If a
configuration specifies a template, the effective configuration of the task will be the result of
merging the two YAML files (configuration file and template). The semantics of this merge operation
is described below. Determined stores this effective configuration so that future changes to a
template will not affect the reproducibility of experiments that used a previous version of the
configuration template.

A single configuration file can use at most one configuration template. A configuration template
cannot itself use another configuration template.

Using Templates to Simplify Experiment Configurations
-----------------------------------------------------

An experiment can use a configuration template by using the ``--template`` command-line option to
specify the name of the desired template.

Here is an example demonstrating how an experiment configuration can be split into a reusable
template and a simplified configuration.

Consider the experiment configuration below:

.. code:: yaml

   name: mnist_tf_const
   checkpoint_storage:
     type: s3
     access_key: my-access-key
     secret_key: my-secret-key
     bucket: my-bucket-name
   data:
     base_url: https://s3-us-west-2.amazonaws.com/determined-ai-datasets/mnist/
     training_data: train-images-idx3-ubyte.gz
     training_labels: train-labels-idx1-ubyte.gz
     validation_set_size: 10000
   hyperparameters:
     base_learning_rate: 0.001
     weight_cost: 0.0001
     global_batch_size: 64
     n_filters1: 40
     n_filters2: 40
   searcher:
     name: single
     metric: error
     max_length:
       batches: 500
     smaller_is_better: true

You may find that the values for the ``checkpoint_storage`` field are the same for many experiments
and you want to use a configuration template to reduce the redundancy. You might write a template
like the following:

.. code:: yaml

   description: template-tf-gpu
   checkpoint_storage:
     type: s3
     access_key: my-access-key
     secret_key: my-secret-key
     bucket: my-bucket-name

Then the experiment configuration for this experiment can be written as below:

.. code:: yaml

   description: mnist_tf_const
   data:
     base_url: https://s3-us-west-2.amazonaws.com/determined-ai-datasets/mnist/
     training_data: train-images-idx3-ubyte.gz
     training_labels: train-labels-idx1-ubyte.gz
     validation_set_size: 10000
   hyperparameters:
     base_learning_rate: 0.001
     weight_cost: 0.0001
     global_batch_size: 64
     n_filters1: 40
     n_filters2: 40
   searcher:
     name: single
     metric: error
     max_length:
       batches: 500
     smaller_is_better: true

To launch the experiment with the template:

.. code:: bash

   $ det experiment create --template template-tf-gpu mnist_tf_const.yaml <model_code>

Using the CLI to Work with Templates
------------------------------------

The :ref:`Determined command-line interface <cli-ug>` can be used to list, create, update, and
delete configuration templates. This functionality can be accessed through the ``det template``
sub-command. This command can be abbreviated as ``det tpl``.

To list all the templates stored in Determined, use ``det template list``. You can also use the
``-d`` or ``--detail`` option to show additional details.

.. code::

   $ det tpl list
   Name
   -------------------------
   template-s3-tf-gpu
   template-s3-pytorch-gpu
   template-s3-keras-gpu

To create or update a template, use ``det tpl set template_name template_file``.

.. code::

   $ cat > template-s3-keras-gpu.yaml << EOL
   description: template-s3-keras-gpu
   checkpoint_storage:
     type: s3
     access_key: my-access-key
     secret_key: my-secret-key
     bucket: my-bucket-name
   EOL
   $ det tpl set template-s3-keras-gpu template-s3-keras-gpu.yaml
   Set template template-s3-keras-gpu

Merge Behavior
--------------

Suppose we have a template that specifies top-level fields ``a`` and ``b`` and a configuration that
specifies fields ``b`` and ``c``. The merged configuration will have fields ``a``, ``b``, and ``c``.
The value for field ``a`` will simply be the value set in the template. Likewise, the value for
field ``c`` will be whatever was specified in the configuration. The final value for field ``b``,
however, depends on the value's type:

-  If the field specifies a scalar value, the merged value will be the one specified by the
   configuration (the configuration overrides the template).

-  If the field specifies a list value, the merged value will be the concatenation of the list
   specified in the template and that specified in the configuration.

   Note that there are exceptions to this rule for ``bind_mounts`` and ``resources.devices``. It may
   be the case that the both the original config and the template will attempt to mount to the same
   ``container_path``, which would result in an unstable config. In those situations, the original
   config is preferred, and the conflicting bind mount or device from the template is omitted in the
   merged result.

-  If the field specifies an object value, the resulting value will be the object generated by
   recursively applying this merging algorithm to both objects.

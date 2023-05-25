.. _multi-gpu-training-concept:

###############################
 Distributed Training Concepts
###############################

*******************************************
 How Determined Distributed Training Works
*******************************************

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

***********************************************
 Reduce Computation and Communication Overhead
***********************************************

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

*************************************************
 How to Train Effectively with Large Batch Sizes
*************************************************

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

***********************************************
 Model Characteristics that Affect Performance
***********************************************

Deep learning models typically perform dense updates, where every model parameter is updated for
every training sample. Because of this, the amount of communication per mini-batch (step 3 in the
distributed training loop) is dependent on the number of model parameters. Models that have fewer
parameters such as `ResNet-50 <https://arxiv.org/pdf/1512.03385.pdf>`_ (~30 million parameters)
train more efficiently in distributed settings than models with more parameters such as `VGG-16
<https://arxiv.org/pdf/1505.06798.pdf>`_ (~136 million parameters). If planning to utilize
distributed training, we encourage users to be mindful of their model size when designing models.

*******************************
 Debug Performance Bottlenecks
*******************************

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

*****************
 Reproducibility
*****************

Determined aims to support *reproducible* machine learning experiments: that is, the result of
running a Determined experiment should be deterministic, so that rerunning a previous experiment
should produce an identical model. For example, this ensures that if the model produced from an
experiment is ever lost, it can be recovered by rerunning the experiment that produced it.

Status
======

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
============

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
=================

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
====================================

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
=================

TensorFlow does not fully support the extraction or restoration of a single, global RNG state.
Consequently, pausing experiments that use a TensorFlow-based framework may introduce an additional
source of entropy.

*******************
 Optimize Training
*******************

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
==============

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
=======

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

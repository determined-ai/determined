.. _optimizing-training:

#####################
 Optimizing Training
#####################

When optimizing the training speed of a model, the first step is to understand where and why
training is slow. Once the bottlenecks have been identified, the next step is to do further
investigation and experimentation to alleviate those bottlenecks.

To understand the performance profile of a training job, the training code and infrastructure need
to be instrumented. Many different layers can be instrumented, from raw throughput all the way down
to GPU kernels.

Determined provides two tools out-of-the-box for instrumenting training:

-  :ref:`System Metrics <how-to-profiling-system-metrics>`: measurements of hardware usage
-  :ref:`Timings <how-to-profiling-timings>`: durations of actions taken during training, such as
   data loading

System Metrics are useful to see if the software is taking full advantage of the available hardware,
particularly around GPU usage, data loading, and network communication during distributed training.
Timings are useful for identifying the section of code to focus on for optimizations. Most commonly,
Timings help answer the question of whether the dataloader is the main bottleneck in training.

.. _how-to-profiling:

.. _how-to-profiling-system-metrics:

****************
 System Metrics
****************

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

*********
 Timings
*********

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
-  ``step_lr_schedulers``: amount of time to update the LR schedules
-  ``from_device``: time to transfer output from device to host
-  ``reduce_metrics``: time taken to calculate global metrics in distributed training

\* ``train_batch`` is typically the forward pass and the backward pass, but it is a user-defined
function so it could include other steps.

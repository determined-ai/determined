.. _profiling:

###########
 Profiling
###########

Optimizing a model's performance is often a trade-off between accuracy, time, and resource
requirements. Training deep learning models is a time and resource intensive process, where each
iteration can take several hours and accumulate heavy hardware costs. Though sometimes this cost is
inherent to the task, unnecessary resource consumption can be caused by suboptimal code or bugs.
Thus, achieving optimal model performance requires an understanding of how your model interacts with
the system's computational resources.

Profiling collects metrics on how computational resources like CPU, GPU, and memory are being
utilized during a training job. It can reveal patterns in resource utilization that indicate
performance bottlenecks and pinpoint areas of the code or pipeline that are causing slowdowns or
inefficiencies.

A training job can be profiled at many different layers, from generic system-level metrics to
individual model operators and GPU kernels. Determined provides a few options for profiling, each
targeting a different layer in a training job at various levels of detail:

-  :ref:`Determined system metrics profiler <how-to-profiling-det-profiler>` collects general
   system-level metrics and provides an overview of hardware usage during an experiment.
-  :ref:`Native profiler integration <how-to-profiling-native-profilers>` enables model profiling in
   training APIs that provides fine-grained metrics specific to your model.
-  :ref:`Prometheus/Grafana integration <how-to-profiling-prom-grafana>` can be set up to track
   detailed hardware metrics and monitor overall cluster health.

.. _how-to-profiling:

.. _how-to-profiling-det-profiler:

*********************
 Determined Profiler
*********************

Determined comes with a built-in profiler that provides out-of-the-box tracking for system-level
metrics. System metrics are statistics around hardware usage, such as GPU utilization, disk usage,
and network throughput.

These metrics provide a general overview of resource usage during a training run, and can be useful
for quickly identifying ineffective usages of computational resources. When the system metrics
reported for an experiment do not match hardware expectations, that is a sign that the software may
be able to be optimized to make better use of the hardware resources.

The Determined profiler collects a set of system metrics throughout an experiment which can be
visualized in the WebUI under the experiment's "Profiler" tab. It is supported for all training
APIs, but is not enabled by default.

Visit :ref:`core-profiler` to find out how to enable and configure the Determined profiler for your
experiment.

The following system metrics are tracked:

-  *GPU utilization (percent)*: utilization of a GPU device
-  *GPU free memory (bytes)*: amount of free memory available on a GPU device
-  *Network throughput - sent (bytes/s)*: bytes sent system-wide
-  *Network throughput (received)*: bytes received system-wide
-  *Disk IOPS (operations/s)*: number of read + writes system-wide
-  *Disk throughput - reads (bytes/s)*: bytes read system-wide
-  *Disk throughput - writes (bytes/s)*: bytes written system-wide
-  *Host available memory (bytes)*: amount of memory available (not including swap) system-wide
-  *CPU utilization (percent)*: utilization of CPU cores, averaged across all cores in the system

For distributed training, these metrics are collected for every agent. The data is broken down by
agent, and GPU metrics can be further broken down by GPU.

.. note::

   System Metrics record agent-level metrics, so when there are multiple experiments on the same
   agent, it is difficult to analyze.

.. _how-to-profiling-native-profilers:

***************************
 Native Training Profilers
***************************

Sometimes system-level profiling doesn't capture enough data to help debug bottlenecks in model
training code. Identifying inefficiencies in individual training operations or steps requires a more
fine-grained context than generic system metrics can provide. For this level of profiling,
Determined supports integration with training profilers that are native to their frameworks:

-  :ref:`PyTorch Profiler <pytorch_profiler>`
-  :ref:`DeepSpeed Profiler <deepspeed-profiler>`
-  :class:`Keras TensorBoard callback <determined.keras.TensorBoard>`

Please see your framework's profiler documentation and the Determined Training API guide for usage
details.

.. _how-to-profiling-prom-grafana:

************************************
 Prometheus and Grafana Integration
************************************

For a more resource-centric view of Determined jobs, Determined provides a Prometheus endpoint along
with a pre-configured Grafana dashboard. These can be set up to track detailed hardware usage
metrics for a Determined cluster, and can be helpful for alerting and monitoring cluster health.

The Prometheus endpoint aggregates system metrics and associates them with Determined concepts such
as experiments, tags, and resource pools, which can be viewed in Grafana. Determined provides a
Grafana dashboard that shows real-time resource metrics across an entire cluster as well as
experiments, containers, and resource pools.

Visit :ref:`configure-prometheus-grafana` to find out how to enable this functionality.

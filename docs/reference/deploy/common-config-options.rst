.. _common-configuration-options:

##############################
 Common Configuration Options
##############################

*************
 Master Port
*************

By default, the master listens on TCP port 8080. This can be configured via the ``port`` option.

.. _security:

**********
 Security
**********

The master can secure all incoming connections using `TLS
<https://en.wikipedia.org/wiki/Transport_Layer_Security>`__. That ability requires a TLS private key
and certificate to be provided; set the options ``security.tls.cert`` and ``security.tls.key`` to
paths to a PEM-encoded TLS certificate and private key, respectively, to do so. If TLS is enabled,
the default port becomes 8443 rather than 8080. See :ref:`tls` for more information.

.. _agent-network-proxy:

*************************************
 Configuring Trial Runner Networking
*************************************

The master is capable of selecting the network interface that trial runners will use to communicate
when performing distributed (multi-machine) training. The network interface can be configured by
editing ``task_container_defaults.dtrain_network_interface``. If left unspecified, which is the
default setting, Determined will auto-discover a common network interface shared by the trial
runners.

.. note::

   For :ref:`multi-gpu-training`, Determined automatically detects a common network interface shared
   by the agent machines. If your cluster has multiple common network interfaces, please specify the
   fastest one.

Additionally, the ports used by the GLOO and NCCL libraries, which are used during distributed
(multi-machine) training can be configured to fall within user-defined ranges. If left unspecified,
ports will be chosen randomly from the unprivileged port range (1024-65535).

****************************
 Default Checkpoint Storage
****************************

See :ref:`checkpoint-storage` for details.

.. _telemetry:

***********
 Telemetry
***********

By default, the master and WebUI collect anonymous information about how Determined is being used.
This usage information is collected so that we can improve the design of the product. Determined
does not report information that can be used to identify individual users of the product, nor does
it include model source code, model architecture/checkpoints, training datasets, training and
validation metrics, logs, or hyperparameter values.

The information we collect from the master periodically includes:

-  a unique, randomly generated ID for the current database and for the current instance of the
   master
-  the version of Determined
-  the version of Go that was used to compile the master
-  the number of registered :ref:`users <users>`
-  the number of experiments that have been created
-  the total number of trials across all experiments
-  the number of active, paused, completed, and canceled experiments
-  whether tasks are scheduled using Kubernetes or the built-in Determined scheduler
-  the total number of slots (e.g., GPUs)
-  the number of slots currently being utilized
-  the type of each configured resource pool

We also record when the following events happen:

-  an experiment is created
-  an experiment changes state
-  an agent connects or disconnects
-  a user is created (the username is not transmitted)

When an experiment is created, we report:

-  the name of the hyperparameter search method
-  the total number of hyperparameters
-  the number of slots (e.g., GPUs) used by each trial in the experiment
-  the name of the container image used

When a task terminates, we report:

-  the start and end time of the task

-  the number of slots (e.g., GPUs) used

-  for experiments, we also report:

   -  the number of trials in the experiment
   -  the total number of training workloads across all trials in the experiment
   -  the total elapsed time for all workloads across all trials in the experiment

The information we collect from the WebUI includes:

-  pages that are visited
-  errors that occur (both network errors and uncaught exceptions)
-  user-triggered actions

To disable telemetry reporting in both the master and the WebUI, start the master with the
``--telemetry-enabled=false`` flag (this can also be done by editing the master config file or
setting an environment variable, as with any other configuration option). Disabling telemetry
reporting will not affect the functionality of Determined in any way.

.. _open_telemetry:

OpenTelemetry
=============

Separate from the telemetry reporting mentioned above, Determined also supports `OpenTelemetry
<https://opentelemetry.io/>`__ to collect traces. This is disabled by default; to enable it, use the
master configuration setting ``telemetry.otel-enabled``. When enabled, the master will send
OpenTelemetry traces to a collector running at ``localhost:4317``. A different endpoint can be set
via the ``telemetry.otel-endpoint`` configuration setting.

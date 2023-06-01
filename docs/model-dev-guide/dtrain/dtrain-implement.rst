.. _multi-gpu-training-implement:

###################################################
 Introduction to Implementing Distributed Training
###################################################

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
training, as described in :ref:`cluster-configuration`.

***************
 Configuration
***************

Slots Per Trial
===============

The ``resources.slots_per_trial`` field in the :ref:`experiment-config-reference` controls the
number of GPUs used to train a single trial.

By default, this field is set to a value of ``1``, which essentially disables distributed training.
If you increase the ``slots_per_trial`` value, this will automatically enable multi-GPU training.
Bear in mind that these GPUs can either be located on a single machine or distributed across
multiple machines. The experiment configuration merely dictates the number of GPUs to be used in the
training process, while the Determined job scheduler decides whether to schedule the task on a
single agent or multiple agents. Whether to schedule the task on a single agent or multiple agents
depends on the machines in the cluster and other active workloads.

Multi-machine parallelism allows you to further parallelize training across more GPUs. To use this
feature, ``slots_per_trial`` should be set as a multiple of the total number of GPUs on an agent
machine. For example, if your resource pool consists of multiple 8-GPU agent machines, valid
``slots_per_trial`` values would be 16, 24, 32, and so on.

In the following configuration, trials will use the combined resources of multiple machines to train
a model:

.. code:: yaml

   resources:
     slots_per_trial: 16  # Two 8-GPU agent machines will be used in a trial

For distributed multi-machine training, Determined will automatically detect a common network
interface that is shared by the agent machines. If your cluster has multiple common network
interfaces, we advise specifying the fastest one in :ref:`cluster-configuration` under
``task_container_defaults.dtrain_network_interface``.

You can reduce computational overhead by setting the ``global_batch_size`` to the largest batch size
that fits into a single GPU multiplied times the number of slots.

When the ``slots_per_trial`` field is set, the per-slot (i.e., per-GPU) batch size is set to
``global_batch_size // slots_per_trial``. The per-slot and global batch sizes can be accessed
through the context using :func:`context.get_per_slot_batch_size()
<determined.TrialContext.get_per_slot_batch_size>` and :func:`context.get_global_batch_size()
<determined.TrialContext.get_global_batch_size>`, respectively. If ``global_batch_size`` is not
evenly divisible by ``slots_per_trial``, the remainder is dropped.

If the value of :ref:`slots_per_trial <exp-config-resources-slots-per-trial>` is greater than the
number of slots available on a single agent, Determined schedules it over multiple machines. When
scheduling a multi-machine distributed training job, Determined requires that all slots (GPUs) on an
agent are used by the job. For example, in a cluster composed of 8-GPU agents, an experiment that
has :ref:`slots_per_trial <exp-config-resources-slots-per-trial>` set to ``12`` will not be
scheduled and will wait indefinitely. For more details, you can visit :ref:`Scheduling Behavior
<dtrain-scheduling>`.

You might have existing tasks that are running on a single machine that are preventing your
multi-GPU trials from acquiring sufficient GPUs. To alleviate this, you may want to consider
adjusting ``slots_per_trial`` or terminating existing tasks to free up slots in your cluster.

Global Batch Size
=================

During distributed training, the ``global_batch_size`` specified in the
:ref:`experiment-config-reference` is partitioned across ``slots_per_trial`` GPUs. The per-GPU batch
size is set to: ``global_batch_size // slots_per_trial``. If ``slots_per_trial`` does not divide
``global_batch_size`` evenly, the remainder is dropped. For convenience, the per-GPU batch size can
be accessed via the Trial API, using :func:`context.get_per_slot_batch_size
<determined.TrialContext.get_per_slot_batch_size>`.

For improved performance, *weak-scaling* is recommended. Weak-scaling means proportionally
increasing your ``global_batch_size`` with ``slots_per_trial``. For example, you might change
``global_batch_size`` and ``slots_per_trial`` from 32 and 1 to 128 and 4, respectively.

Note that adjusting ``global_batch_size`` can impact your model convergence, which in turn can
affect your training and/or testing accuracy. You might need to adjust model hyperparameters like
the learning rate, or consider using a different optimizer when training with larger batch sizes.

.. _multi-gpu-training-implement-adv-optimizations:

Advanced Optimizations
======================

Determined supports optimizations to further reduce training time.

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

To learn more about these optimizations, visit the ``optimizations`` section in the
:ref:`experiment-config-reference`.

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
downloaded data is stored. A convenient identifier is the ``rank`` of the current process: the
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
compute resources required for a job are available before the job is launched. When using
distributed training, take note of the following details related to scheduler behavior:

-  If ``slots_per_trial`` is less than or equal to the number of slots on a single agent, Determined
   considers scheduling multiple distributed training jobs on a single agent. This approach is
   designed to improve utilization and to allow multiple small training jobs to run on a single
   agent. For example, an agent with eight GPUs could be assigned two 4-GPU jobs or four 2-GPU jobs.

-  On the other hand, if ``slots_per_trial`` is greater than the number of slots on a single agent,
   Determined schedules the distributed training job onto multiple agents. A multi-machine
   distributed training job is only scheduled onto an agent if it results in utilizing all of the
   agent GPUs. This strategy ensures good performance and utilizes the full network bandwidth of
   each machine while minimizing inter-machine networking. For example, if all the agents in your
   cluster have eight GPUs each, you should submit jobs with ``slots_per_trial`` set to a multiple
   of eight, such as 8, 16, or 24.

.. warning::

   If these scheduling constraints for multi-machine distributed training are not satisfied,
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

.. _config-template:

*************************
 Configuration Templates
*************************

In a typical organization, many Determined configuration files will share similar settings. This can
cause redundancy. For example, all training workloads run at a given organization might use the same
checkpoint storage configuration. One way to reduce this redundancy is to use *configuration
templates*. This feature allows users to consolidate settings shared across many experiments into a
single YAML file that can be referenced by configurations needings those settings.

Each configuration template has a unique name and is stored by the Determined master. If a
configuration employs a template, the effective configuration of the task will be the outcome of
merging the two YAML files (the configuration file and the template). The semantics of this merge
operation are described below. Determined stores this effective configuration to ensure future
changes to a template do not affect the reproducibility of experiments that used a previous version
of the configuration template.

A single configuration file can use at most one configuration template. A configuration template
cannot itself use another configuration template.

Leveraging Templates to Simplify Experiment Configurations
==========================================================

An experiment can adopt a configuration template by using the ``--template`` command-line option to
denote the name of the desired template.

The following example demonstrates splitting an experiment configuration into a reusable template
and a simplified configuration.

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

You may find that many experiments share the same values for the ``checkpoint_storage`` field,
leading to redundancy. To reduce the redundancy you could use a configuration template. For example,
consider the following template:

.. code:: yaml

   description: template-tf-gpu
   checkpoint_storage:
     type: s3
     access_key: my-access-key
     secret_key: my-secret-key
     bucket: my-bucket-name

The experiment configuration for this experiment can then be written using the following code:

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

Managing Templates through the CLI
==================================

The :ref:`Determined command-line interface <cli-ug>` provides tools for managing configuration
templates including listing, creating, updating, and deleting templates. This functionality can be
accessed through the ``det template`` sub-command. This command can be abbreviated as ``det tpl``.

To list all the templates stored in Determined, use ``det template list``. To show additional
details, use the ``-d`` or ``--detail`` option.

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
==============

To demonstrate merge behavior when merging a template and a configuration, let's say we have a
template that specifies top-level fields ``a`` and ``b``, and a configuration that specifies fields
``b`` and ``c``. The resulting merged configuration will have fields ``a``, ``b``, and ``c``. The
value for field ``a`` will simply be the value set in the template. Likewise, the value for field
``c`` will be whatever was specified in the configuration. The final value for field ``b``, however,
depends on the value's type:

-  If the field specifies a scalar value, the configuration's value will take precedence in the
   merged configuration (overriding the template's value).

-  If the field specifies a list value, the merged value will be the concatenation of the list
   specified in the template and the one specified in the configuration.

   .. note::

      There are certain exceptions for ``bind_mounts`` and ``resources.devices``. There could be
      situations where both the original config and the template will attempt to mount to the same
      ``container_path``, resulting in an unstable configuration. In such scenarios, the original
      configuration is preferred, and the conflicting bind mount or device from the template is
      omitted in the merged result.

-  If the field specifies an object value, the resulting value will be the object generated by
   recursively applying this merging algorithm to both objects.

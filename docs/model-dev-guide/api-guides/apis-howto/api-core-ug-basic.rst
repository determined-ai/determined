.. _api-core-ug-basic:

###########################
 Get Started with Core API
###########################

Learn how to get started with the Core API by incrementing a single integer in a loop.

.. note::

   You can also visit the :ref:`api-core-ug` where you'll how to adapt model training code to use
   the Core API that uses the PyTorch MNIST model as an example.

+------------------------------------------------------------------+
| Visit the API reference                                          |
+==================================================================+
| :ref:`core-reference`                                            |
+------------------------------------------------------------------+

With the Core API you can train arbitrary models on the Determined platform with seamless access to
the the following capabilities:

-  metrics tracking
-  checkpoint tracking and preemption support
-  hyperparameter search
-  distributing work across multiple GPUs and/or nodes

These are the same features provided by the higher-level PyTorchTrial, DeepSpeedTrial, and
TFKerasTrial APIs: those APIs are implemented using the Core API.

This user guide shows you how to get started using the Core API.

************************
 Get the Tutorial Files
************************

Access the tutorial files via the :download:`core_api.tgz </examples/core_api.tgz>` download or
directly from the `Github repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.

*****************
 Getting Started
*****************

As a simple introduction, this example training script increments a single integer in a loop,
instead of training a model with machine learning. The changes shown for the example model should be
similar to the changes you make in your actual model.

The ``0_start.py`` training script used in this example contains your simple "model":

.. literalinclude:: ../../../../examples/tutorials/core_api/0_start.py
   :language: python
   :start-at: import

To run this script, create a configuration file with at least the following values:

.. literalinclude:: ../../../../examples/tutorials/core_api/0_start.yaml
   :language: yaml

The actual configuration file can have any name, but this example uses ``0_start.yaml``.

Run the code using the command:

.. code:: bash

   det e create 0_start.yaml . -f

If you navigate to this experiment in the WebUI no metrics are displayed because you have not yet
reported them to the master using the Core API.

.. _core-metrics:

****************
 Report Metrics
****************

The Core API makes it easy to report training and validation metrics to the master during training
with only a few new lines of code.

#. For this example, create a new training script called ``1_metrics.py`` by copying the
   ``0_start.py`` script from :ref:`core-getting-started`.

#. Begin by importing import the ``determined`` module:

   .. literalinclude:: ../../../../examples/tutorials/core_api/1_metrics.py
      :language: python
      :start-after: NEW: import determined
      :end-before: def main

#. Enable ``logging``, using the ``det.LOG_FORMAT`` for logs. This enables useful log messages from
   the ``determined`` library, and ``det.LOG_FORMAT`` enables filter-by-level in the WebUI.

   .. literalinclude:: ../../../../examples/tutorials/core_api/1_metrics.py
      :language: python
      :start-at: logging.basicConfig
      :end-at: logging.error

#. In your ``if __name__ == "__main__"`` block, wrap the entire execution of ``main()`` within the
   scope of :meth:`determined.core.init`, which prepares resources for training and cleans them up
   afterward. Add the ``core_context`` as a new argument to ``main()`` because the Core API is
   accessed through the ``core_context`` object.

   .. literalinclude:: ../../../../examples/tutorials/core_api/1_metrics.py
      :language: python
      :start-at: with det.core.init

#. Within ``main()``, add two calls: (1) report training metrics periodically during training and
   (2) report validation metrics every time a validation runs.

   .. literalinclude:: ../../../../examples/tutorials/core_api/1_metrics.py
      :language: python
      :pyobject: main

   The ``report_validation_metrics()`` call typically happens after the validation step, however,
   actual validation is not demonstrated by this example.

#. Create a ``1_metrics.yaml`` file with an ``entrypoint`` invoking the new ``1_metrics.py`` file.
   You can copy the ``0_start.yaml`` configuration file and change the first couple of lines:

   .. literalinclude:: ../../../../examples/tutorials/core_api/1_metrics.yaml
      :language: yaml
      :lines: 1-2

#. Run the code using the command:

   .. code:: bash

      det e create 1_metrics.yaml . -f

#. You can now navigate to the new experiment in the WebUI and view the plot populated with the
   training and validation metrics.

The complete ``1_metrics.py`` and ``1_metrics.yaml`` listings used in this example can be found in
the :download:`core_api.tgz </examples/core_api.tgz>` download or in the `Github repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.

.. _core-checkpoints:

********************
 Report Checkpoints
********************

By checkpointing periodically during training and reporting those checkpoints to the master, you can
stop and restart training in two different ways: either by pausing and reactivating training using
the WebUI, or by clicking the Continue Trial button after the experiment completes.

These two types of continuations have different behaviors. While you always want to preserve the
value you are incrementing (the "model weight"), you do not always want to preserve the batch index.
When you pause and reactivate you want training to continue from the same batch index, but when
starting a fresh experiment you want training to start with a fresh batch index. You can save the
trial ID in the checkpoint and use it to distinguish the two types of continues.

#. Create a new ``2_checkpoints.py`` training script called by copying the ``1_metrics.py`` script
   from :ref:`core-metrics`.

#. Write save and load methods for your model:

   .. literalinclude:: ../../../../examples/tutorials/core_api/2_checkpoints.py
      :language: python
      :pyobject: save_state

   .. literalinclude:: ../../../../examples/tutorials/core_api/2_checkpoints.py
      :language: python
      :pyobject: load_state

#. In your ``if __name__ == "__main__"`` block, use the ClusterInfo API to gather additional
   information about the task running on the cluster, specifically a checkpoint to load from and the
   trial ID, which you also pass to ``main()``.

   .. literalinclude:: ../../../../examples/tutorials/core_api/2_checkpoints.py
      :language: python
      :start-at: info = det.get_cluster_info()

   It is recommended that you always follow this pattern of extracting values from the ClusterInfo
   API and passing the values to lower layers of your code, instead of accessing the ClusterInfo API
   directly in the lower layers. In this way the lower layer can be written to run on or off of the
   Determined cluster.

#. Within ``main()``, add logic to continue from a checkpoint, when a checkpoint is provided:

   .. literalinclude:: ../../../../examples/tutorials/core_api/2_checkpoints.py
      :language: python
      :start-at: def main
      :end-at: for batch in range(starting_batch, 100)

#. You can checkpoint your model as frequently as you like. For this exercise, save a checkpoint
   after each training report, and check for a preemption signal after each checkpoint:

   .. literalinclude:: ../../../../examples/tutorials/core_api/2_checkpoints.py
      :language: yaml
      :dedent:
      :start-at: if steps_completed % 10 == 0
      :end-before: core_context.train.report_validation_metrics

#. Create a ``2_checkpoints.yaml`` file by copying the ``0_start.yaml`` file and changing the first
   couple of lines:

   .. literalinclude:: ../../../../examples/tutorials/core_api/2_checkpoints.yaml
      :language: yaml
      :lines: 1-2

#. Run the code using the command:

   .. code:: bash

      det e create 2_checkpoints.yaml . -f

#. You can navigate to the experiment in the WebUI and pause it mid-training. The trial shuts down
   and stops producing logs. If you reactivate training it resumes where it stopped. After training
   is completed, click Continue Trial to see that fresh training is started but that the model
   weight continues from where previous training finished.

The complete ``2_checkpoints.py`` and ``2_checkpoints.yaml`` listings used in this example can be
found in the :download:`core_api.tgz </examples/core_api.tgz>` download or in the `Github repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.

.. _core-hpsearch:

***********************
 Hyperparameter Search
***********************

With the Core API you can run advanced hyperparameter searches with arbitrary training code. The
hyperparameter search logic is in the master, which coordinates many different Trials. Each trial
runs a train-validate-report loop:

.. table::

   +----------+--------------------------------------------------------------------------+
   | Train    | Train until a point chosen by the hyperparameter search algorithm and    |
   |          | obtained via the Core API.  The length of training is absolute, so you   |
   |          | have to keep track of how much you have already trained to know how much |
   |          | more to train.                                                           |
   +----------+--------------------------------------------------------------------------+
   | Validate | Validate your model to obtain the metric you configured in the           |
   |          | ``searcher.metric`` field of your experiment config.                     |
   +----------+--------------------------------------------------------------------------+
   | Report   | Use the Core API to report results to the master.                        |
   +----------+--------------------------------------------------------------------------+

#. Create a ``3_hpsearch.py`` training script by copying the ``2_checkpoints.py`` script you created
   in :ref:`core-checkpoints`.

#. In your ``if __name__ == "__main__"`` block, access the hyperparameter values chosen for this
   trial using the ClusterInfo API and configure the training loop accordingly:

   .. literalinclude:: ../../../../examples/tutorials/core_api/3_hpsearch.py
      :language: python
      :dedent:
      :start-at: hparams = info.trial.hparams

#. Modify ``main()`` to run the train-validate-report loop mentioned above by iterating through
   ``core_context.searcher.operations()``. Each :class:`~determined.core.SearcherOperation` from
   :meth:`~determined.core.SearcherContext.operations` has a ``length`` attribute that specifies the
   absolute length of training to complete. After validating, report the searcher metric value using
   ``op.report_completed()``.

   .. literalinclude:: ../../../../examples/tutorials/core_api/3_hpsearch.py
      :language: python
      :dedent:
      :start-at: batch = starting_batch
      :end-at: op.report_completed

#. Because the training length can vary, you might exit the train-validate-report loop before saving
   the last of your progress. To handle this, add a conditional save after the loop ends:

   .. literalinclude:: ../../../../examples/tutorials/core_api/3_hpsearch.py
      :language: python
      :dedent:
      :start-at: if last_checkpoint_batch != steps_completed
      :end-at: save_state

#. Create a new ``3_hpsearch.yaml`` file and add an ``entrypoint`` that invokes ``3_hpsearch.py``:

   .. literalinclude:: ../../../../examples/tutorials/core_api/3_hpsearch.yaml
      :language: yaml
      :lines: 1-2

   Add a ``hyperparameters`` section with the integer-type ``increment_by`` hyperparameter value
   that referenced in the training script:

   .. literalinclude:: ../../../../examples/tutorials/core_api/3_hpsearch.yaml
      :language: yaml
      :start-at: hyperparameters:
      :end-at: maxval

#. Run the code using the command:

   .. code:: bash

      det e create 3_hpsearch.yaml . -f

The complete ``3_hpsearch.py`` and ``3_hpsearch.yaml`` listings used in this example can be found in
the :download:`core_api.tgz </examples/core_api.tgz>` download or in the `Github repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.

.. _core-distributed:

**********************
 Distributed Training
**********************

The Core API has special considerations for running distributed training. Some of the more important
considerations are:

-  Access to all IP addresses of every node in the Trial (through the ClusterInfo API).

-  Communication primitives such as :meth:`~determined.core.DistributedContext.allgather`,
   :meth:`~determined.core.DistributedContext.gather`, and
   :meth:`~determined.core.DistributedContext.broadcast` to give you out-of-the-box coordination
   between workers.

-  Since many distributed training frameworks expect all workers in training to operate in-step, the
   :meth:`~determined.core.PreemptContext.should_preempt` call is automatically synchronized across
   workers so that all workers decide to preempt or continue as a unit.

#. Create a ``4_distributed.py`` training script by copying the ``3_hpsearch.py`` from
   :ref:`core-hpsearch`.

#. Add launcher logic to execute one worker subprocess per slot.

   Start with a ``launcher_main()`` function that executes one worker subprocess per slot.

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :pyobject: launcher_main

   Typically, you do not have to write your own launcher. Determined provides launchers for Horovod,
   ``torch.distributed``, and DeepSpeed. Additionally, there are third-party launchers available,
   such as ``mpirun``. When using a custom or third-party launcher, wrap your worker script in the
   ``python -m determined.launcher.wrap_rank`` wrapper script so the WebUI log viewer can filter
   logs by rank.

   Also add a ``worker_main()`` that will run training on each slot:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :pyobject: worker_main

   Then modify your ``if __name__ == "__main__"`` block to invoke the correct ``*_main()`` based on
   command-line arguments:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-at: slots_per_node = len(info.slot_ids)

#. In the training code, use the ``allgather`` primitive to do a "distributed" increment, to gain
   experience using the communication primitives:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-at: all_increment_bys =
      :end-at: x += sum(all_increment_bys)

#. Usually, trial logs are easier to read when status is only printed on the chief worker:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-after: some logs are easier to read
      :end-at: logging.info

#. Only the chief worker is permitted to report training metrics, report validation metrics, upload
   checkpoints, or report searcher operations completed. This rule applies to the steps you take
   periodically during training:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-at: if steps_completed % 10 == 0
      :end-at: return

   The rule also applies to the steps you take after validating:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-after: only the chief may report validation metrics
      :end-at: op.report_completed

   The rule also applies to the conditional save after the main loop completes:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-at: again, only the chief may upload checkpoints
      :end-at: save_state

#. Create a ``4_distributed.yaml`` file by copying the ``3_distributed.yaml`` file and changing the
   first couple of lines:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.yaml
      :language: yaml
      :lines: 1-2

   Set the ``resources.slots_per_trial`` field to the number of GPUs you want:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.yaml
      :language: yaml
      :start-at: resources:
      :end-at: slots_per_trial:

   You can return to using the ``single`` searcher instead of an ``adaptive_asha`` hyperparameter
   search:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.yaml
      :language: yaml
      :start-at: searcher:
      :end-at: max_length:

#. Run the code using the Determined CLI with the following command:

   .. code:: bash

      det e create 4_distributed.yaml . -f

The complete ``4_distributed.py`` and ``3_hpsearch.yaml`` listings used in this example can be found
in the :download:`core_api.tgz </examples/core_api.tgz>` download or in the `Github repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.

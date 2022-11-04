.. _core-checkpoints:

####################
 Report Checkpoints
####################

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

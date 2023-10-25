.. _save-load-checkpoints:

#######################################
 Save and Load State Using Checkpoints
#######################################

.. meta::
   :description: Learn how to utilize detached mode to save and load states via checkpoints.

Leveraging :ref:`detached mode <detached-mode-index>`, you can easily save the state at a particular
point during training and restore it when needed. This is especially useful for resuming training
after interruptions or failures.

For the full script, visit the `GitHub repository
<https://github.com/determined-ai/determined/blob/main/examples/features/unmanaged/2_checkpoints.py>`_.

************
 Objectives
************

These step-by-step instructions walk you through:

-  Initializing the core context with checkpoint storage
-  Loading the latest checkpoint
-  Resuming sending metrics to the trial
-  Saving checkpoints periodically

After completing this guide, you will be able to:

-  Understand how checkpoints operate in detached mode
-  Implement state-saving and restoration in your training routine
-  Use the Core API to handle checkpoints effectively

***************
 Prerequisites
***************

**Required**

-  A Determined cluster

**Recommended**

-  :ref:`simple-metrics-reporting`

*************************************************************
 Step 1: Initialize the Core Context with Checkpoint Storage
*************************************************************

To begin, you need to set up the core context, specifying the checkpoint storage path. If recovering
from a failure, an external experiment and trial ID can be used to identify which artifact to log
metrics to:

.. code:: python

   def main():
       core_v2.init(
           defaults=core_v2.DefaultConfig(
               name="unmanaged-2-checkpoints",
               checkpoint_storage="/path/to/checkpoint",
           ),
           unmanaged=core_v2.UnmanagedConfig(
               external_experiment_id="my-existing-experiment",
               external_trial_id="my-existing-trial",
           ),
       )

************************************
 Step 2: Load the Latest Checkpoint
************************************

Fetch the latest checkpoint and load it:

.. code:: python

   latest_checkpoint = core_v2.info.latest_checkpoint
   initial_i = 0
   if latest_checkpoint is not None:
       with core_v2.checkpoint.restore_path(latest_checkpoint) as path:
           with (path / "state").open() as fin:
               ckpt = fin.read()
               i_str, _ = ckpt.split(",")
               initial_i = int(i_str)

*********************************************
 Step 3: Resume Sending Metrics to the Trial
*********************************************

Continue logging metrics to the trial from where you left off:

.. code:: python

   for i in range(initial_i, initial_i + 100):
       loss = random.random()
       print(f"training loss is: {loss}")
       core_v2.train.report_training_metrics(steps_completed=i, metrics={"loss": loss})

***************************************
 Step 4: Save Checkpoints Periodically
***************************************

Store a new checkpoint every 10 steps:

.. code:: python

   if (i + 1) % 10 == 0:
       with core_v2.checkpoint.store_path({"steps_completed": i}) as (path, uuid):
           with (path / "state").open("w") as fout:
               fout.write(f"{i},{loss}")

End your training script and close the core context:

.. code:: python

   core_v2.close()

Navigate to ``<DET_MASTER_IP:PORT>`` in your web browser to see the experiment.

************
 Next Steps
************

Having walked through this guide, you now understand how to effectively use checkpoints in detached
mode. Try more examples using detached mode or learn more about Determined by visiting the
:ref:`tutorials <tutorials-index>`.

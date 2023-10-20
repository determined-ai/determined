.. _distributed-training-checkpointing:

#####################################################
 Use Distributed Training with Sharded Checkpointing
#####################################################

.. meta::
   :description: Discover how to employ detached mode for distributed training with sharded checkpointing. This guide will illustrate the necessary steps.

In this tutorial, we'll show you how to manage sharded checkpoints using :ref:`detached mode
<detached-mode-index>`.

We will guide you through a process that includes setting up torch distributed for distributed
training, sharding data between different processes, and saving sharded checkpoints.

************
 Objectives
************

These step-by-step instructions will cover:

-  Initializing communications library and distributed context
-  Implementing sharding for batches across processes
-  Reporting training and validation metrics
-  Storing sharded checkpoints
-  Running code with torch distributed and the appropriate cluster topology arguments

By the end of this guide, you'll:

-  Grasp how distributed training functions in detached mode
-  Know how to shard checkpoints effectively
-  Understand how to employ the core API for managing distributed training sessions

***************
 Prerequisites
***************

**Required**

-  A Determined cluster
-  Torch library for distributed training

**Recommended**

-  `Simple Metrics Reporting User Guide <simple-metrics-reporting>`_
-  `Save and Load State using Checkpoints User Guide <save-load-checkpoints>`_

*******************************************************************
 Step 1: Initialize Communications Library and Distributed Context
*******************************************************************

Import necessary libraries and set up the distributed context:

.. code:: python

   import logging
   import torch.distributed as dist
   import determined
   import determined.core
   from determined.experimental import core_v2

   def main():
       dist.init_process_group("gloo")
       distributed = core_v2.DistributedContext.from_torch_distributed()
       core_v2.init(
           defaults=core_v2.DefaultConfig(
               name="unmanaged-3-torch-distributed",
           ),
           distributed=distributed,
       )

****************************************
 Step 2: Shard Batches Across Processes
****************************************

Distribute data between processes and report metrics:

.. code:: python

   size = dist.get_world_size()
   for i in range(100):
       if i % size == dist.get_rank():
           core_v2.train.report_training_metrics(
               steps_completed=i,
               metrics={"loss": random.random(), "rank": dist.get_rank()},
           )

************************************************
 Step 3: Report Validation Metrics Periodically
************************************************

.. code:: python

   if (i + 1) % 10 == 0:
       core_v2.train.report_validation_metrics(
           steps_completed=i,
           metrics={"loss": random.random(), "rank": dist.get_rank()},
       )

***********************************
 Step 4: Store Sharded Checkpoints
***********************************

.. code:: python

   ckpt_metadata = {"steps_completed": i, f"rank_{dist.get_rank()}": "ok"}
   with core_v2.checkpoint.store_path(ckpt_metadata, shard=True) as (path, uuid):
       with (path / f"state_{dist.get_rank()}").open("w") as fout:
           fout.write(f"{i},{dist.get_rank()}")

*******************************************************
 Step 5: Retrieve Web Server Address and Close Context
*******************************************************

.. code:: python

   if dist.get_rank() == 0:
       print(
           "See the experiment at:",
           core_v2.url_reverse_webui_exp_view(),
       )
   core_v2.close()

*****************************************
 Step 6: Run Code with Torch Distributed
*****************************************

Execute your code with the following command, setting appropriate parameters for your cluster
topology:

.. code:: bash

   python3 -m torch.distributed.run --nnodes=1 --nproc_per_node=2 \
    --master_addr 127.0.0.1 --master_port 29400 --max_restarts 0 \
    my_torch_disributed_script.py

************
 Next Steps
************

Now that you've mastered the art of using detached mode for distributed training with sharded
checkpointing, you're poised to scale your training sessions and tackle larger datasets and more
complex models.

.. _pytorch-trainer-guide:

#####################
 PyTorch Trainer API
#####################

This guide will help you get up and running with the PyTorch Trainer API.

+-----------------------------------------------------------------------------+
| Visit the API reference                                                     |
+=============================================================================+
| :doc:`/reference/reference-training/training/api-pytorch-trainer-reference` |
+-----------------------------------------------------------------------------+

With the PyTorch Trainer API, you can implement and iterate on model training code locally before
running on cluster. When you are satisfied with your model code, you configure and submit the code
on cluster.

The PyTorch Trainer API lets you do the following:

-  Work locally, iterating on your model code.
-  Debug models in your favorite debug environment (e.g., directly on your machine, IDE, or Jupyter
   notebook).
-  Run training scripts without needing to use an experiment configuration file.
-  Load previous saved checkpoints directly into your model.

************
 Objectives
************

After completing the steps in this guide, you will be able to do the following:

-  Define a PyTorch Trial
-  Initialize the PyTorch Trial and the Trainer
-  Run Your Training Script Locally
-  Submit Your Trial for Training On Cluster
-  Load checkpoints

***************
 Prerequisites
***************

-  Access to a Determined cluster. If you have not yet installed Determined, refer to the
   :ref:`installation instructions <install-cluster>`.

-  The Determined CLI should be installed on your local machine. For installation instructions,
   visit :ref:`Commands and Shells: Installation <install-cli>`. After installing the CLI, configure
   it to connect to your Determined cluster by setting the ``DET_MASTER`` environment variable to
   the hostname or IP address where Determined is running.

********************************
 Step 1: Define a PyTorch Trial
********************************

Start by defining a PyTorchTrial by instantiating the ``Trial`` and ``TrialContext`` objects. When
using the PyTorch Trainer API, you do not have to initialize and wrap models and optimizers inside
of the ``Trial.__init__`` method. If desired, pass a wrapped model to ``Trial.__init__``.

.. code:: python

   class MyPyTorchTrial(pytorch.PyTorchTrial):
       def __init__(self, context: PyTorchTrialContext, hparams: Dict) -> None:
           self.context = context
           self.model = context.wrap_model(nn.Sequential(
               nn.Linear(9216, 128),
           ))
           self.optimizer = context.wrap_optimizer(torch.optim.Adadelta(
               self.model.parameters(), lr=hparams["lr"])
           )

       def train_batch(
               self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
       ) -> Dict[str, torch.Tensor]:
           ...
           output = self.model(data)
           loss = torch.nn.functional.nll_loss(output, labels)

           self.context.backward(loss)
           self.context.step_optimizer(self.optimizer)

           return {"loss": loss}

       def evaluate_batch(self, batch: pytorch.TorchData) -> Dict[str, Any]:
           ...
           return {"validation_loss": validation_loss, "accuracy": accuracy}

       def build_training_data_loader(self) -> DataLoader:
           ...
           return DataLoader(train_set)

       def build_validation_data_loader(self) -> DataLoader:
           ...
           return DataLoader(validation_set)

******************************************************
 Step 2: Initialize the PyTorch Trial and the Trainer
******************************************************

After defining the PyTorch Trial, initialize the trial and the trainer.

.. code:: python

   from determined import pytorch

   def main():
       # pytorch.init() returns a PyTorchTrialContext for instantiating PyTorchTrial
       with det.pytorch.init() as train_context:
           trial = MyPyTorchTrial(train_context)
           trainer = det.pytorch.Trainer(trial, train_context)

           # (Optional) Configure Determined profiler before calling .fit()
           trainer.configure_profiler(enabled=True,
                                      sync_timings=True,
                                      begin_on_batch=0,
                                      end_after_batch=10)

           # Train
           trainer.fit(
               checkpoint_period=pytorch.Batch(10),
               validation_period=pytorch.Batch(10),
           )

   if __name__ == "__main__":
       # Configure logging here instead of through the expconf
       logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
       main()

******************************************
 Step 3: Run Your Training Script Locally
******************************************

With the PyTorch Trainer API, you can run training scripts locally without needing to use an
experiment configuration file. Be sure to specify max_length in the ``.fit()`` call, and
global_batch_size in pytorch.init().

Run this script directly (python3 train.py), or inside of a Jupyter notebook.

.. code:: python

   + hparams = {"global_batch_size": 32, "lr": 0.02}
   + expconf = yaml.safe_load(pathlib.Path("./det.yaml").read_text())

   # hparams and exp_conf are optional. Only needed by init() if training code calls
   # context.get_hparams() or context.get_experiment_config()
   + with det.pytorch.init(hparams=hparams, exp_conf=expconf) as train_context:
         # (Optional) Preferred way to access hparams in the Trial
   +     trial = MyPytorchTrial(train_context, hparams)
         trainer = det.pytorch.Trainer(trial, train_context)
         trainer.fit(
   +         max_length=pytorch.Epoch(1),
             checkpoint_period=pytorch.Batch([2,5]),
             validation_period=pytorch.Batch(10),
       )

Local + Distributed Training
============================

Local training can utilize multiple GPUs on a single node with a few modifications to the above
code.

.. note::

   Both Horovod and PyTorch Distributed backends are supported.

.. code:: python

    def main():
   +     # Initialize distributed backend before pytorch.init()
   +     dist.init_process_group(backend="gloo|nccl")

   +     # Set flag used by internal PyTorch training loop
   +     os.environ["USE_TORCH_DISTRIBUTED"] = "true"

   +     # Initialize DistributedContext specifying chief IP
         with det.pytorch.init(
   +       distributed=core.DistributedContext.from_torch_distributed (chief_ip="localhost")
         ) as train_context:
             trial = MNistTrial(train_context)
             trainer = det.pytorch.Trainer(trial, train_context)
             trainer.fit(
                 max_length=pytorch.Epoch(1),
                 checkpoint_period=pytorch.Batch(10),
                 validation_period=pytorch.Batch(10),
             )

Call your distributed backend's launcher directly: ``torchrun --nproc_per_node=4 train.py``.

Local Training - Test Mode
==========================

PyTorch Trainer accepts a test_mode parameter which, if true, trains and validates your training
code for only one batch, checkpoints, then exits. This is helpful for debugging code or writing 
automated tests around your model code.

.. code:: python

    trainer.fit(
                 max_length=pytorch.Epoch(1),
                 checkpoint_period=pytorch.Batch(10),
                 validation_period=pytorch.Batch(10),
   +             # Train and validate 1 batch, then checkpoint and exit.
   +             test_mode=True
             )

This replaces the legacy test mode codepath, which supports this functionality for trials 
going through harness:

.. code:: bash

   det e create det.yaml . --local --test
 
 

**************************************************************************
 Step 4: Prepare Your Training Code for Deploying to a Determined Cluster
**************************************************************************

Once you are satisfied with the results of training the model locally, you submit the code to a
cluster.

**Example workflow of frequent iterations between local debugging and cluster deployment**

This code should allow for local and cluster training with no code changes.

.. code:: python

    def main():
   +   local = det.get_cluster_info() is None
   +   if local:
   +       # (Optional) Initialize distributed backend before pytorch.init()
   +       dist.init_process_group(backend="gloo|nccl")
   +       # Set flag used by internal PyTorch training loop
   +       os.environ["USE_TORCH_DISTRIBUTED"] = "true"
   +       distributed_context = core.DistributedContext.from_torch_distributed (chief_ip="localhost")
   +   else:
   +       distributed_context = None

   +     with det.pytorch.init(
   +       distributed=distributed_context
         ) as train_context:
             trial = MNistTrial(train_context)
             trainer = det.pytorch.Trainer(trial, train_context)
             trainer.fit(
                 max_length=pytorch.Epoch(1),
                 checkpoint_period=pytorch.Batch(10),
                 validation_period=pytorch.Batch(10),
                 latest_checkpoint=det.get_cluster_info().latest_checkpoint
             )

**To run Trainer API solely on-cluster, the code is much simpler**

.. code:: python

   def on_cluster():
       """
       On-cluster training with Trainer API (entrypoint: python3 train.py)
       """
       hparams = det.get_cluster_info().trial.hparams

       with det.pytorch.init() as train_context:
           trial_inst = model.MNistTrial(train_context, hparams)
           trainer = det.pytorch.Trainer(trial_inst, train_context)
           trainer.fit(
               checkpoint_period=pytorch.Batch(10),
               validation_period=pytorch.Batch(100),
               latest_checkpoint=det.get_cluster_info().latest_checkpoint,
           )

***************************************************
 Step 5: Submit Your Trial for Training on Cluster
***************************************************

To run your experiment on cluster, you'll need to create an experiment configuration (YAML) file.
Your experiment configuration file must contain searcher configuration and entrypoint.

.. note::

   ``global_batch_size`` is required if ``max_length`` is configured in records

.. code:: python

   name: my_pytorch_trainer_trial
   hyperparameters:
     global_batch_size: 32
   searcher:
     name: single
     metric: validation_loss
     max_length:
       batches: 937
   resources:
     slots_per_trial: 8
   entrypoint: python3 -m determined.launch.torch_distributed python3 train.py

Submit the trial to the cluster:

.. code:: bash

   det e create det.yaml .

*****************************
 Step 6: Loading Checkpoints
*****************************

To load a checkpoint from a checkpoint saved using Trainer, you'll need to download the checkpoint
to a file directory and use an import helper method to import modules. You should instantiate your
loaded Trial with a ``CheckpointLoadContext``.

``det.import_from_path`` allows you to import from a specific directory and cleans up afterwards.
Even if you are importing identically-named files, you can import them as separate modules. This is
intended to help when you have, for example, a current model_def.py, but also import an older
model_def.py from a checkpoint into the same interpreter, without conflicts (so long as you import
them as different names, of course).

``CheckpointLoadContext`` is a special PyTorchTrialContext that can be used to load Trial classes
outside of normal training loops. It does not support any training features such as metrics
reporting or uploading checkpoints and is intended for use with the Trainer directly.

.. code:: python

   import determined as det
   from determined import pytorch
   from determined.experimental import client
    # Download checkpoint and load training code from checkpoint.
       path = client.get_checkpoint(MY_CHECKPOINT_UUID)
       with det.import_from_path(path + "/code/"):
           import my_model_def

   # Create CheckpointLoadContext for instantiating trial.
   context = pytorch.CheckpointLoadContext()
   # Instantiate trial with context and any other args.
   my_trial = my_model_def.MyTrial(context, ...)

*********
 Summary
*********

By following the steps in this guide, you were able to iterate on and debug your model training code
locally before running on cluster.

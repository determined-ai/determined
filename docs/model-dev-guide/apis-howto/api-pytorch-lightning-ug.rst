:orphan:

#######################
 PyTorch Lightning API
#######################

.. meta::
   :description: Discover how to use the PyTorch Lightning API to train a PyTorch Lightning model in Determined. It includes step-by-step instructions for installation and usage, as well as sample code snippets and tips.

.. attention::

   The PyTorch Lightning API has been deprecated. The information on this page will still work on
   Determined version 0.23.3 or earlier and will stop working in 0.23.4. If you are using PyTorch
   Lightning, you are advised to migrate to the :ref:`Core API <core-getting-started>`.

   This page will be removed in a future version of the documentation.

In this guide, you'll learn how to use the PyTorch Lightning API.

+-------------------------------------------------------------------------------+
| Visit the API reference                                                       |
+===============================================================================+
| :doc:`/reference/training/api-pytorch-lightning-reference`                    |
+-------------------------------------------------------------------------------+

This document guides you through training a PyTorch Lightning model in Determined. You need to
implement a trial class that inherits :class:`~determined.pytorch.lightning.LightningAdapter` and
specify it as the entrypoint in the :doc:`experiment configuration
</reference/training/experiment-config-reference>`.

PyTorch Lightning Adapter, defined here as ``LightningAdapter``, provides a quick way to train your
PyTorch Lightning models with all the Determined features, such as mid-epoch preemption, easy
distributed training, simple job submission to the Determined cluster, and so on.

LightningAdapter is built on top of our :doc:`PyTorch API
</model-dev-guide/apis-howto/api-pytorch-ug>`, which has a built-in training loop that integrates
with the Determined features. However, it only supports `LightningModule
<https://pytorch-lightning.readthedocs.io/en/stable/common/lightning_module.html>`_ (v1.2.0). To
migrate your code from the `Trainer
<https://pytorch-lightning.readthedocs.io/en/stable/common/trainer.html>`_, please read more about
:doc:`PyTorch API </model-dev-guide/apis-howto/api-pytorch-ug>` and
:ref:`experiment-config-reference`.

*****************************
 Port PyTorch Lightning Code
*****************************

Porting your ``PyTorchLightning`` code is often pretty simple:

#. Bring in your ``LightningModule`` and ``LightningDataModule`` and initialize them
#. Create a new trial based on ``LightningAdapter`` and initialize it.
#. Define the dataloaders.

Here is an example:

.. code:: python

   from determined.pytorch import PyTorchTrialContext, DataLoader
   from determined.pytorch.lightning import LightningAdapter

   # bring in your LightningModule and optionally LightningDataModule
   from mnist import LightningMNISTClassifier, MNISTDataModule


   class MNISTTrial(LightningAdapter):
       def __init__(self, context: PyTorchTrialContext) -> None:
           # instantiate your LightningModule with hyperparameter from the Determined
           # config file or from the searcher for automatic hyperparameter tuning.
           lm = LightningMNISTClassifier(lr=context.get_hparam("learning_rate"))

           # instantiate your LightningDataModule and make it distributed training ready.
           data_dir = f"/tmp/data-rank{context.distributed.get_rank()}"
           self.dm = MNISTDataModule(context.get_data_config()["url"], data_dir)

           # initialize LightningAdapter.
           super().__init__(context, lightning_module=lm)
           self.dm.prepare_data()

       def build_training_data_loader(self) -> DataLoader:
           self.dm.setup()
           dl = self.dm.train_dataloader()
           return DataLoader(
               dl.dataset, batch_size=dl.batch_size, num_workers=dl.num_workers
           )

       def build_validation_data_loader(self) -> DataLoader:
           self.dm.setup()
           dl = self.dm.val_dataloader()
           return DataLoader(
               dl.dataset, batch_size=dl.batch_size, num_workers=dl.num_workers
           )

In this approach, the ``LightningModule`` is not paired with the PyTorch Lightning ``Trainer`` so
that there are some methods and hooks that are not supported. Read about those here:

-  No separate test-set definition in Determined: ``test_step``, ``test_step_end``,
   ``test_epoch_end``, ``on_test_batch_start``, ``on_test_batch_end``, ``on_test_epoch_start``,
   ``on_test_epoch_end``, ``test_dataloader``.

-  No fit or pre-train stage: ``setup``, ``teardown``, ``on_fit_start``, ``on_fit_end``,
   ``on_pretrain_routine_start``, ``on_pretrain_routine_end``.

-  Additionally, no: ``training_step_end`` & ``validation_step_end``, ``hiddens`` parameter in
   ``training_step`` and ``tbptt_split_batch``, ``transfer_batch_to_device``,
   ``get_progress_bar_dict``, ``on_train_epoch_end``, ``manual_backward``, ``backward``,
   ``optimizer_step``, ``optimizer_zero_grad``

In addition, we also patched some ``LightningModule`` methods to make porting your code easier:

-  ``log`` and ``log_dict`` are patched to always ship their values to TensorBoard. In the current
   version only the first two arguments in ``log``: ``key`` and ``value``, and the first argument in
   ``log_dict`` are supported.

.. note::

   Make sure to return the metric you defined as ``searcher.metric`` in your experiment's
   :ref:`configuration <experiment-config-reference>` from your ``validation_step``.

.. note::

   Determined will automatically log the metrics you return from ``training_step`` and
   ``validation_step`` to TensorBoard.

.. note::

   Determined environment images no longer contain PyTorch Lightning. To use PyTorch Lightning, add
   a line similar to the following in the ``startup-hooks.sh`` script:

.. code:: bash

   pip install pytorch_lightning==1.5.10 torchmetrics==0.5.1

To learn about this API, start by reading the trial definitions from the following examples:

-  :download:`gan_mnist_pl.tgz </examples/gan_mnist_pl.tgz>`
-  :download:`mnist_pl.tgz </examples/mnist_pl.tgz>`

***********
 Load Data
***********

.. note::

   Before loading data, read this document :doc:`/model-dev-guide/load-model-data` to understand how
   to work with different sources of data.

Loading your dataset when using PyTorch Lightning works the same way as it does with :doc:`PyTorch
API </model-dev-guide/apis-howto/api-pytorch-ug>`.

If you already have a ``LightningDataModule`` you can bring it in and use it to implement
``build_training_data_loader`` and ``build_validation_data_loader`` methods easily. For more
information read PyTorchTrial's section on Data Loading.

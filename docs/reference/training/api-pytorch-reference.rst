###############################
 ``det.pytorch`` API Reference
###############################

.. meta::
   :description: Familiarize yourself with the det.pytorch API. This PyTorch-based training loop includes the PyTorchTrial class, the PyTorchTrialContext class, and the Trainer class.

+--------------------------------------------+
| User Guide                                 |
+============================================+
| :ref:`pytorch_trial_ug`                    |
+--------------------------------------------+

Determined offers a PyTorch-based training loop that is fully integrated with the Determined
platform which includes:

-  :class:`~determined.pytorch.PyTorchTrial`, which you must subclass to define things like model
   architecture, optimizer, data loaders, and how to train or validate a single batch.
-  :class:`~determined.pytorch.PyTorchTrialContext`, which can be accessed from within
   ``PyTorchTrial`` and contains runtime methods used for training with the ``PyTorch`` API.
-  :class:`~determined.pytorch.Trainer`, which is used for customizing and executing the training
   loop around a ``PyTorchTrial``.

.. _pytorch_api_ref:

*************************************
 ``determined.pytorch.PyTorchTrial``
*************************************

.. autoclass:: determined.pytorch.PyTorchTrial
   :members:
   :member-order: bysource
   :special-members: __init__

********************************************
 ``determined.pytorch.PyTorchTrialContext``
********************************************

.. autoclass:: determined.pytorch.PyTorchTrialContext
   :members:

********************************************************
 ``determined.pytorch.PyTorchTrialContext.distributed``
********************************************************

.. autoclass:: determined.core._distributed.DistributedContext
   :members:
   :inherited-members:
   :member-order: bysource
   :noindex:

***************************************************
 ``determined.pytorch.PyTorchExperimentalContext``
***************************************************

.. autoclass:: determined.pytorch.PyTorchExperimentalContext
   :members:
   :exclude-members: reduce_metrics, reset_reducers, wrap_reducer

.. _pytorch-dataloader:

***********************************
 ``determined.pytorch.DataLoader``
***********************************

.. autoclass:: determined.pytorch.DataLoader
   :members:

************************************
 ``determined.pytorch.LRScheduler``
************************************

.. autoclass:: determined.pytorch.LRScheduler
   :members:
   :special-members: __init__

********************************
 ``determined.pytorch.Reducer``
********************************

.. autoclass:: determined.pytorch.Reducer
   :members:

.. _pytorch-metric-reducer:

**************************************
 ``determined.pytorch.MetricReducer``
**************************************

.. autoclass:: determined.pytorch.MetricReducer
   :members: reset, per_slot_reduce, cross_slot_reduce
   :member-order: bysource

.. _pytorch-callbacks:

****************************************
 ``determined.pytorch.PyTorchCallback``
****************************************

.. autoclass:: determined.pytorch.PyTorchCallback
   :members:

********************************************************
 ``determined.pytorch.load_trial_from_checkpoint_path``
********************************************************

.. autofunction:: determined.pytorch.load_trial_from_checkpoint_path

********************************
 ``determined.pytorch.Trainer``
********************************

.. autoclass:: determined.pytorch.Trainer
   :members:

*******************************
 ``determined.pytorch.init()``
*******************************

.. autofunction:: determined.pytorch.init

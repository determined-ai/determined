###############
 API Reference
###############

*************************************
 ``determined.pytorch.PyTorchTrial``
*************************************

.. autoclass:: determined.pytorch.PyTorchTrial
   :members:
   :inherited-members:
   :member-order: bysource
   :special-members: __init__

********************************************
 ``determined.pytorch.PyTorchTrialContext``
********************************************

.. autoclass:: determined.pytorch.PyTorchTrialContext
   :members:
   :inherited-members:
   :show-inheritance:

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

.. _pytorch-samplers:

*********************************
 ``determined.pytorch.samplers``
*********************************

.. automodule:: determined.pytorch.samplers
   :members:

.. _pytorch-callbacks:

****************************************
 ``determined.pytorch.PyTorchCallback``
****************************************

.. autoclass:: determined.pytorch.PyTorchCallback
   :members:

.. _pytorch-writer:

***************************************************************
 ``determined.tensorboard.metric_writers.pytorch.TorchWriter``
***************************************************************

.. autoclass:: determined.tensorboard.metric_writers.pytorch.TorchWriter

********************************************************
 ``determined.pytorch.load_trial_from_checkpoint_path``
********************************************************

.. autofunction:: determined.pytorch.load_trial_from_checkpoint_path

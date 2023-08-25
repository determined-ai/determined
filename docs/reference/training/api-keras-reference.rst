#############################
 ``det.keras`` API Reference
#############################

+-------------------------------------------------+
| User Guide                                      |
+=================================================+
| :ref:`api-keras-ug`                             |
+-------------------------------------------------+

***********************************
 ``determined.keras.TFKerasTrial``
***********************************

.. autoclass:: determined.keras.TFKerasTrial
   :members:
   :exclude-members: trial_controller_class, trial_context_class
   :inherited-members:
   :member-order: bysource
   :special-members: __init__

******************************************
 ``determined.keras.TFKerasTrialContext``
******************************************

.. autoclass:: determined.keras.TFKerasTrialContext
   :members: wrap_model, wrap_dataset, wrap_optimizer, configure_fit
   :member-order: bysource
   :inherited-members:

   TFKerasTrialContext always has a :class:`~determined.core._distributed.DistributedContext`
   accessible via ``context.distributed`` for information related to distributed training.

   TFKerasTrialContext always has a :class:`~determined.keras.TFKerasExperimentalContext` accessible
   via ``context.experimental`` for information related to experimental features.

******************************************************
 ``determined.keras.TFKerasTrialContext.distributed``
******************************************************

.. autoclass:: determined.core._distributed.DistributedContext
   :members:
   :inherited-members:
   :member-order: bysource
   :noindex:

*************************************************
 ``determined.keras.TFKerasExperimentalContext``
*************************************************

.. autoclass:: determined.keras.TFKerasExperimentalContext
   :members: cache_train_dataset, cache_validation_dataset
   :member-order: bysource
   :show-inheritance:

********************************
 ``determined.keras.callbacks``
********************************

.. autoclass:: determined.keras.callbacks.Callback
   :members:

.. autoclass:: determined.keras.callbacks.EarlyStopping

.. autoclass:: determined.keras.callbacks.ReduceLROnPlateau

.. autoclass:: determined.keras.callbacks.TensorBoard

******************************************************
 ``determined.keras.load_model_from_checkpoint_path``
******************************************************

.. autoclass:: determined.keras.load_model_from_checkpoint_path
   :members:

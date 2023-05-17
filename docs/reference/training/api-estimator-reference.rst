#################################
 ``det.estimator`` API Reference
#################################

.. warning::

   ``EstimatorTrial`` is deprecated and will be removed in a future version. TensorFlow has advised
   Estimator users to switch to Keras since TensorFlow 2.0 was released. Consequently, we recommend
   users of EstimatorTrial to switch to the :class:`~determined.keras.TFKerasTrial` class.

+-----------------------------------------------------+
| User Guide                                          |
+=====================================================+
| :doc:`/model-dev-guide/apis-howto/api-estimator-ug` |
+-----------------------------------------------------+

*****************************************
 ``determined.estimator.EstimatorTrial``
*****************************************

.. autoclass:: determined.estimator.EstimatorTrial
   :members:
   :exclude-members: trial_controller_class
   :inherited-members:
   :member-order: bysource
   :special-members: __init__

************************************************
 ``determined.estimator.EstimatorTrialContext``
************************************************

.. autoclass:: determined.estimator.EstimatorTrialContext
   :members:
   :inherited-members:
   :member-order: bysource
   :show-inheritance:

   EstimatorTrialContext always has a :class:`~determined.core._distributed.DistributedContext`
   accessible via ``context.distributed`` for information related to distributed training.

   EstimatorTrialContext always has a :class:`~determined.estimator.EstimatorExperimentalContext`
   accessible via ``context.experimental`` for information related to experimental features.

************************************************************
 ``determined.estimator.EstimatorTrialContext.distributed``
************************************************************

.. autoclass:: determined.core._distributed.DistributedContext
   :members:
   :inherited-members:
   :member-order: bysource
   :noindex:

*******************************************************
 ``determined.estimator.EstimatorExperimentalContext``
*******************************************************

.. autoclass:: determined.estimator.EstimatorExperimentalContext
   :members: cache_train_dataset, cache_validation_dataset
   :member-order: bysource

****************************************
 ``determined.estimator.MetricReducer``
****************************************

.. autoclass:: determined.estimator.MetricReducer
   :members: accumulate, cross_slot_reduce
   :member-order: bysource

**********************************
 ``determined.estimator.RunHook``
**********************************

.. autoclass:: determined.estimator.RunHook
   :members: on_checkpoint_load, on_checkpoint_end, on_trial_close

**************************************************************
 ``determined.estimator.load_estimator_from_checkpoint_path``
**************************************************************

.. autoclass:: determined.estimator.load_estimator_from_checkpoint_path
   :members:

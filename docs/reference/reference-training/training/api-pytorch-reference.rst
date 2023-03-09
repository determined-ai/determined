.. _pytorch-api-reference:

#######################
 PyTorch API Reference
#######################

+--------------------------------------------+
| User Guide                                 |
+============================================+
| :doc:`/training/apis-howto/api-pytorch-ug` |
+--------------------------------------------+

***************
 PyTorch Trial
***************

``determined.pytorch.PyTorchTrial``
===================================

.. autoclass:: determined.pytorch.PyTorchTrial
   :members:
   :inherited-members:
   :member-order: bysource
   :special-members: __init__

``determined.pytorch.PyTorchTrialContext``
==========================================

.. autoclass:: determined.pytorch.PyTorchTrialContext
   :members:
   :inherited-members:
   :show-inheritance:

``determined.pytorch.PyTorchTrialContext.distributed``
======================================================

.. autoclass:: determined.core._distributed.DistributedContext
   :members:
   :inherited-members:
   :member-order: bysource
   :noindex:

``determined.pytorch.PyTorchExperimentalContext``
=================================================

.. autoclass:: determined.pytorch.PyTorchExperimentalContext
   :members:
   :exclude-members: reduce_metrics, reset_reducers, wrap_reducer

.. _pytorch-dataloader:

``determined.pytorch.DataLoader``
=================================

.. autoclass:: determined.pytorch.DataLoader
   :members:

``determined.pytorch.LRScheduler``
==================================

.. autoclass:: determined.pytorch.LRScheduler
   :members:
   :special-members: __init__

``determined.pytorch.Reducer``
==============================

.. autoclass:: determined.pytorch.Reducer
   :members:

.. _pytorch-metric-reducer:

``determined.pytorch.MetricReducer``
====================================

.. autoclass:: determined.pytorch.MetricReducer
   :members: reset, per_slot_reduce, cross_slot_reduce
   :member-order: bysource

.. _pytorch-samplers:

``determined.pytorch.samplers``
===============================

.. automodule:: determined.pytorch.samplers
   :members:

.. _pytorch-callbacks:

``determined.pytorch.PyTorchCallback``
======================================

.. autoclass:: determined.pytorch.PyTorchCallback
   :members:

.. _pytorch-writer:

``determined.tensorboard.metric_writers.pytorch.TorchWriter``
=============================================================

.. autoclass:: determined.tensorboard.metric_writers.pytorch.TorchWriter

``determined.pytorch.load_trial_from_checkpoint_path``
======================================================

.. autofunction:: determined.pytorch.load_trial_from_checkpoint_path

*****************
 PyTorch Trainer
*****************

.. code:: python

   determined.pytorch.init(
       *,
       hparams: Optional[Dict] = None,
       exp_conf: Optional[Dict[str, Any]] = None,
       distributed: Optional[core.DistributedContext] = None,
       aggregation_frequency: int = 1,
   ) -> pytorch.PyTorchTrialContext:
    """
    Creates a PyTorchTrialContext for use with a PyTorchTrial. All trainer.* calls must be within
    the scope of this context because there are resources started in __enter__ that must be
    cleaned up in __exit__.

    Arguments:
        hparams: (Optional) instance of hyperparameters for the trial
        exp_conf: (Optional) for local-training mode. If unset, calling
            context.get_experiment_config() will fail.
        distributed: (Optional) custom distributed training configuration
        aggregation_frequency: number of batches before gradients are exchanged in distributed
            training. This value is configured here because it is used in context.wrap_optimizer.
    """

``determined.pytorch.Trainer``
==============================

``class determined.pytorch.Trainer(trial: pytorch.PyTorchTrial, context:
pytorch.PyTorchTrialContext)``

``pytorch.Trainer`` is an abstraction on top of a vanilla PyTorch training loop that handles
   many training details under-the-hood, and exposes APIs for configuring training-related features
   such as automatic checkpointing, validation, profiling, metrics reporting, etc.

``Trainer`` must be initialized and called from within a ``pytorch.PyTorchTrialContext``.

.. code:: python

   classmethod configure_profiler(
       sync_timings: bool, enabled: bool, begin_on_batch: int, end_after_batch: int
   ) -> None:

        """
        Configures the Determined profiler. This method should only be called before .fit(), and
        only once within the scope of init(). If called multiple times, the last call's
        configuration will be used.

        Arguments:
            sync_timings: Specifies whether Determined should wait for all GPU kernel streams
                before considering a timing as ended. Defaults to ‘true’. Applies only for
                frameworks that collect timing metrics (currently just PyTorch).
            enabled: Defines whether profiles should be collected or not. Defaults to false.
            begin_on_batch: Specifies the batch on which profiling should begin.
            end_after_batch: Specifies the batch after which profiling should end.
        """

.. code:: python

   classmethod fit(
       checkpoint_period: Optional[pytorch.TrainUnit] = None,
       validation_period: Optional[pytorch.TrainUnit] = None,
       max_length: Optional[pytorch.TrainUnit] = None,
       reporting_period: pytorch.TrainUnit = pytorch.Batch(100),
       checkpoint_policy: str = "best",
       latest_checkpoint: Optional[str] = None,
       step_zero_validation: bool = False,
       test_mode: bool = False,
   ) -> None:

        """
        ``fit()`` trains a ``PyTorchTrial`` configured from the ``Trainer`` and handles checkpointing
        and validation steps, and metrics reporting.

        Arguments:
            checkpoint_period: The number of steps to train for before checkpointing. This is
                a ``TrainUnit`` type (``Batch`` or ``Epoch``) which can take an ``int`` or
                instance of ``collections.abc.Container`` (list, tuple, etc.). For example,
                ``Batch(100)`` would checkpoint every 100 batches, while ``Batch([5, 30, 45])``
                would checkpoint after every 5th, 30th, and 45th batch.
            validation_period: The number of steps to train for before validating. This is a
                ``TrainUnit`` type (``Batch`` or ``Epoch``) which can take an ``int`` or instance
                of ``collections.abc.Container`` (list, tuple, etc.). For example, ``Batch(100)``
                would validate every 100 batches, while ``Batch([5, 30, 45])`` would validate
                after every 5th, 30th, and 45th batch.
            max_length: The maximum number of steps to train for. This value is required and
                only applicable in local training mode. For on-cluster training, this value will
                be ignored; the searcher’s ``max_length`` must be configured from the experiment
                configuration. This is a ``TrainUnit`` type (``Batch`` or ``Epoch``) which takes an
                ``int``. For example, ``Epoch(1)`` would train for a maximum length of one epoch.
                reporting_period:
            checkpoint_policy: Controls how Determined performs checkpoints after validation
                operations, if at all. Should be set to one of the following values:
                    best (default): A checkpoint will be taken after every validation operation
                    that performs better than all previous validations for this experiment.
                    Validation metrics are compared according to the metric and smaller_is_better
                    options in the searcher configuration. This option is only supported for
                    on-cluster training.
                    all: A checkpoint will be taken after every validation, no matter the
                    validation performance.
                    none: A checkpoint will never be taken due to a validation. However,
                    even with this policy selected, checkpoints are still expected to be taken
                    after the trial is finished training, due to cluster scheduling decisions,
                    before search method decisions, or due to min_checkpoint_period.
            latest_checkpoint: Configures the checkpoint used to start or continue training.
                This value should be set to ``det.get_cluster_info().latest_checkpoint`` for
                standard continue training functionality.
            step_zero_validation: Configures whether or not to perform an initial validation
                before training.
            test_mode: Runs a minimal loop of training for testing and debugging purposes. Will
                train and validate one batch. Defaults to false.
        """

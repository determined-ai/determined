###############################
 PyTorch Trainer API Reference
###############################

+----------------------------------------------------+
| User Guide                                         |
+====================================================+
| :doc:`/training/apis-howto/api-pytorch-trainer-ug` |
+----------------------------------------------------+

.. code::

   determined.pytorch.init(
       *,
       hparams: Optional[Dict] = None,
       exp_conf: Optional[Dict[str, Any]] = None,
       distributed: Optional[core.DistributedContext] = None
   ) -> pytorch.PyTorchTrialContext:

``pytorch.init()`` builds a ``pytorch.PyTorchTrialContext`` for use with ``PyTorchTrial``.

Always use this method to construct a ``PyTorchTrialContext`` instead of instantiating the class
directly.

All of the arguments are optional, but ``hparams`` and ``exp_conf`` are used to set the
corresponding variables on ``PyTorchTrialContext``. So if not passed in, calling
``context.get_hparams()`` or ``context.get_experiment_config()`` will fail in local-training mode.
``DistributedContext`` can be optionally passed in to manually configure distributed training;
otherwise, it will be automatically initialized from the launch layer.

All ``trainer.*`` calls must be within the scope of this ``with pytorch.init() as trial_context``,
as there are resources necessary for training which start in the **enter** method and must be
cleaned up in the corresponding **exit**\ () method.

********************************
 ``determined.pytorch.Trainer``
********************************

``class determined.pytorch.Trainer(trial: pytorch.PyTorchTrial, context:
pytorch.PyTorchTrialContext)``

``Trainer`` is the main class for Trainer API. It has the following required arguments: - ``trial``:
an instance of the ``PyTorchTrial`` class - ``context``: the ``PyTorchTrialContext`` returned from
``pytorch.init()``

.. code::

   classmethod configure_profiler(
       sync_timings: bool, enabled: bool, begin_on_batch: int, end_after_batch: int
   ) -> None:

Configure profiler settings. This method can only be called once per ``Trainer`` object and must be
called before ``.fit()``

.. code::

   classmethod fit(
       checkpoint_period: Optional[pytorch.TrainUnit] = None,
       validation_period: Optional[pytorch.TrainUnit] = None,
       max_length: Optional[pytorch.TrainUnit] = None,
       reporting_period: Optional[pytorch.TrainUnit] = None,
       aggregation_frequency: Optional[int] = None,
       checkpoint_policy: Optional[str] = None,
       test_mode: Optional[bool] = None,
   )

``fit()`` trains a ``PyTorchTrial`` configured from the ``Trainer`` and handles checkpointing and
validation steps, and metrics reporting.

``checkpoint_period`` The number of steps to train for before checkpointing. This is a ``TrainUnit``
type (``Batch`` or ``Epoch``) which can take an ``int`` or instance of ``collections.abc.Container``
(list, tuple, etc.). For example, ``Batch(100)`` would checkpoint every 100 batches, while
``Batch([5, 30, 45])`` would checkpoint after every 5th, 30th, and 45th batch.

``validation_period`` The number of steps to train for before validating. This is a ``TrainUnit``
type (``Batch`` or ``Epoch``) which can take an ``int`` or instance of ``collections.abc.Container``
(list, tuple, etc.). For example, ``Batch(100)`` would validate every 100 batches, while ``Batch([5,
30, 45])`` would validate after every 5th, 30th, and 45th batch.

``max_length`` The maximum number of steps to train for. This value is required and only applicable
in local training mode. For on-cluster training, this value will be ignored; the searcher’s
``max_length`` must be configured from the experiment configuration. This is a ``TrainUnit`` type
(``Batch`` or ``Epoch``) which takes an ``int``. For example, ``Epoch(1)`` would train for a maximum
lenght of one epoch.

``reporting_period`` The number of steps to train for before reporting metrics. Note that metrics
are automatically reported before every validation and checkpoint, so this configures additional
metrics reporting outside of those steps. This is a ``TrainUnit`` type (``Batch`` or ``Epoch``)
which can take an ``int`` or instance of ``collections.abc.Container`` (list, tuple, etc.). For
example, ``Batch(100)`` would report metrics every 100 batches, while ``Batch([5, 30, 45])`` would
report after every 5th, 30th, and 45th batch.

``aggregation_frequency`` The number of batches trained before gradients are exchanged during
distributed training. If unset, will default to 1.

``checkpoint_policy`` Controls how Determined performs checkpoints after validation operations, if
at all. Should be set to one of the following values:

best (default): A checkpoint will be taken after every validation operation that performs better
than all previous validations for this experiment. Validation metrics are compared according to the
metric and smaller_is_better options in the searcher configuration. This option is only supported
for on-cluster training.

all: A checkpoint will be taken after every validation, no matter the validation performance.

none: A checkpoint will never be taken due to a validation. However, even with this policy selected,
checkpoints are still expected to be taken after the trial is finished training, due to cluster
scheduling decisions, before search method decisions, or due to min_checkpoint_period.

``test_mode`` Runs a minimal loop of training for testing and debugging purposes. Will train and
validate one batch. Defaults to false.

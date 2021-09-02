import logging
from typing import Any, Callable, Dict, Optional, Union, cast

from determined import pytorch, util

# AMP is only available in PyTorch 1.6+
try:
    import torch.cuda.amp as amp
except ImportError:
    # A warning is logged in _pytorch_context.py
    pass


class PyTorchExperimentalContext:
    def __init__(self, parent: Any) -> None:
        self._parent = parent
        self._auto_amp = False
        self._data_repro_checks_disabled = False
        self._auto_to_device = True

    def use_amp(self) -> None:
        """
        Handles all operations for the most simple cases automatically with a default gradient
        scaler. Specifically, wraps forward pass in an autocast context, scales loss before
        backward pass, unscales before clipping gradients, uses scaler when stepping
        optimizer(s), and updates scaler afterwards. Do not call ``wrap_scaler`` directly when
        using this method.

        PyTorch 1.6 or greater is required for this feature.
        """
        self._parent.wrap_scaler(amp.GradScaler())  # type: ignore
        self._auto_amp = True

    def disable_dataset_reproducibility_checks(self) -> None:
        """
        ``disable_dataset_reproducibility_checks()`` allows you to return an arbitrary
        ``DataLoader`` from :meth:`~determined.pytorch.PyTorchTrial.build_training_data_loader` or
        :meth:`~determined.pytorch.PyTorchTrial.build_validation_data_loader`.

        Normally you would be required to return a ``det.pytorch.DataLoader`` instead, which would
        guarantee that an appropriate ``Sampler`` is used that ensures:

        - When ``shuffle=True``, the shuffle is reproducible.
        - The dataset will start at the right location, even after pausing/continuing.
        - Proper sharding is used during distributed training.

        However, there may be cases where either reproducibility of the dataset is not needed or
        where the nature of the dataset may cause the ``det.pytorch.DataLoader`` to be unsuitable.

        In those cases, you may call ``disable_dataset_reproducibility_checks()`` and you will be
        free to return any ``torch.utils.data.DataLoader`` you like.  Dataset reproducibility will
        still be possible, but it will be your responsibility.  If desired, you may find the
        ``Sampler`` classes in :mod:`determined.pytorch.samplers` to be helpful.
        """

        self._data_repro_checks_disabled = True
        logging.info("disabled dataset reproducibility checks")

    def disable_auto_to_device(self) -> None:
        """
        Prevent the PyTorchTrialController from automatically moving batched data to device.
        Call this if you want to override the default behavior of moving all items of a list,
        tuple, and/or dict to the GPU. Then, you can control how data is moved to the GPU directly
        in the ``train_batch`` and ``evaluate_batch`` methods of your PyTorchTrial definition.
        You should call context.to_device on primitive data types that you do want to move to GPU
        as in the example below.

        .. code-block:: python

            # PyTorchTrial methods.
            def __init__(context): # PyTorchTrial init
                self.context.experimental.disable_auto_to_device()
                ...

            def train_batch(self, context, batch):
                for k, item in batch.items():
                    if k == "img":
                        batch["img"] = self.context.to_device(batch["img"])
                ...
        """
        self._auto_to_device = False
        logging.info("disabled automatically moving data to device")

    @util.deprecated(
        "context.experimental.reset_reducers() is deprecated since 0.15.2 and will be removed in a "
        "future version; use context.reset_reducers() directly."
    )
    def reset_reducers(self) -> None:
        self._parent.reset_reducers()

    @util.deprecated(
        "context.experimental.wrap_reducer() is deprecated since 0.15.2 and will be removed in a "
        "future version; use context.wrap_reducer() directly."
    )
    def wrap_reducer(
        self,
        reducer: Union[Callable, pytorch.MetricReducer],
        name: Optional[str] = None,
        for_training: bool = True,
        for_validation: bool = True,
    ) -> pytorch.MetricReducer:
        return cast(
            pytorch.MetricReducer,
            self._parent.wrap_reducer(reducer, name, for_training, for_validation),
        )

    @util.deprecated(
        "context.experimental.reduce_metrics() is deprecated since 0.15.2 and will be removed in a "
        "future version; use context.reduce_metrics() directly."
    )
    def reduce_metrics(self, for_training: bool) -> Dict[str, Any]:
        return cast(
            dict,
            self._parent.reduce_metrics(for_training),
        )

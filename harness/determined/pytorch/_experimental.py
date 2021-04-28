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

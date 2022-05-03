"""
Convert the LRUpdaterHook in mmcv to a PyTorchCallback.
See: https://github.com/open-mmlab/mmcv/blob/master/mmcv/runner/hooks/lr_updater.py.
"""
from typing import Any, Dict, Optional, cast

import mmcv
import mmcv.runner.hooks as mmcv_hooks

import determined.pytorch as det_torch


class DummyDataloader:
    def __init__(self, epoch_len: Optional[int]):
        self.epoch_len = epoch_len

    def __len__(self) -> Optional[int]:
        return self.epoch_len


class FakeRunner:
    """
    Mocks a mmcv runner and implements the same properties accessed by `LrUpdaterHook`.
    Instead, we get them from the PyTorchTrialContext.
    """

    def __init__(self, context: det_torch.PyTorchTrialContext):
        self.context = context
        self._data_loader = None  # type: Optional[DummyDataloader]
        experiment_config = context.get_experiment_config()
        self.max_length = experiment_config["searcher"]["max_length"]

    @property
    def optimizer(self) -> Dict[int, Any]:
        return {i: opt for i, opt in enumerate(self.context.optimizers)}

    @property
    def data_loader(self) -> Optional[DummyDataloader]:
        # The MMCV lr_updater uses runner.data_loader to get epoch length.
        # We will use a fake data_loader here to return the epoch length.
        if self._data_loader is None:
            self._data_loader = DummyDataloader(self.context._epoch_len)
        return self._data_loader

    @property
    def iter(self) -> Optional[int]:
        return self.context._current_batch_idx

    @property
    def epoch(self) -> Optional[int]:
        return self.context.current_train_epoch()

    @property
    def max_epoch(self) -> int:
        if "epochs" in self.max_length:
            return int(self.max_length["epochs"])
        raise KeyError("max_length is not specified in terms of epochs")

    @property
    def max_iters(self) -> int:
        if "batches" in self.max_length:
            return int(self.max_length["batches"])
        raise KeyError("max_length is not specified in terms of iterations")


def build_lr_hook(lr_config: Dict[Any, Any]) -> mmcv_hooks.LrUpdaterHook:
    assert "policy" in lr_config, "policy must be specified in lr_config"
    policy_type = lr_config.pop("policy")
    if policy_type == policy_type.lower():
        policy_type = policy_type.title()
    hook_type = policy_type + "LrUpdaterHook"
    lr_config["type"] = hook_type
    hook = mmcv.build_from_cfg(lr_config, mmcv_hooks.HOOKS)
    return hook


class LrUpdaterCallback(det_torch.PyTorchCallback):
    """
    Updates the learning rate for optimizers according to the configured LrUpdaterHook.

    mmcv's LrUpdaterHook replaces lr schedulers to perform lr warmup and annealing.
    See: https://github.com/open-mmlab/mmcv/blob/master/mmcv/runner/hooks/lr_updater.py
    for supported lr updaters.

    We mock the behavior of the mmcv hook with our PyTorchCallback.  We do not have a
    `on_batch_start` callback so that is called manually in the `train_batch` method
    of `MMDetTrial`.

    The LrUpaterHook is configured from the mmdet configuration file passed to the
    experiment config.
    """

    def __init__(
        self,
        context: det_torch.PyTorchTrialContext,
        hook: Optional[mmcv_hooks.LrUpdaterHook] = None,
        lr_config: Optional[Dict[Any, Any]] = None,
    ):
        """
        Creates the callback from either the provided `hook` or `lr_config`.
        One of `hook` or `lr_config` must be defined. If both are provided,
        `hook` takes precedence.

        Arguments:
            context: PyTorchTrialContext used to get iterations and epoch information
            hook (Optional): already created mmcv LrUpdaterHook
            lr_config (Optional): configuration for LrUpdaterHook
        """
        self.runner = FakeRunner(context)
        assert (
            hook is not None or lr_config is not None
        ), "One of hook or lr_config must be provided."
        if hook is None:
            lr_config = cast(Dict[Any, Any], lr_config)
            hook = build_lr_hook(lr_config)
        self.hook = hook

    def on_training_start(self) -> None:
        self.hook.before_run(self.runner)

    def on_training_epoch_start(self, epoch_idx: int) -> None:
        self.hook.before_train_epoch(self.runner)

    def on_batch_start(self) -> None:
        self.hook.before_train_iter(self.runner)

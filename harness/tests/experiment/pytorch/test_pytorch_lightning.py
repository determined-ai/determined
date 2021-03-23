import pathlib
import typing
from typing import Any, Dict

import pytest
import torch

import determined as det
from determined import pytorch, workload
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import lightning_adapter_onevar_model as la_model


def check_equal_structures(a: typing.Any, b: typing.Any) -> None:
    """
    Check that two objects, consisting of any nested structures of lists and
    dicts, with leaf values of tensors or built-in objects, are equal in
    structure and values.
    """
    if isinstance(a, dict):
        assert isinstance(b, dict)
        assert len(a) == len(b)
        for key in a:
            assert key in b
            check_equal_structures(a[key], b[key])
    elif isinstance(a, list):
        assert isinstance(b, list)
        assert len(a) == len(b)
        for x, y in zip(a, b):
            check_equal_structures(x, y)
    elif isinstance(a, torch.Tensor):
        assert isinstance(b, torch.Tensor)
        assert torch.allclose(a, b)
    else:
        assert a == b


def fork_trial_override_lm(overrides: dict):
    new_lm_cls = type("new_lm_cls", (la_model.OneVarLM,), overrides)

    def init(self, context):
        super(self.__class__, self).__init__(context, new_lm_cls)

    new_trial_cls = type("new_trial", (la_model.OneVarTrial,), {"__init__": init})
    return new_trial_cls


class TestPyTorchTrial:
    def setup_method(self) -> None:
        # This training setup is not guaranteed to converge in general,
        # but has been tested with this random seed.  If changing this
        # random seed, verify the initial conditions converge.
        self.trial_seed = 17
        self.hparams = {
            "hidden_size": 2,
            "learning_rate": 0.5,
            "global_batch_size": 4,
            "lr_scheduler_step_mode": pytorch.LRScheduler.StepMode.MANUAL_STEP.value,
        }

    def test_checkpointing_and_restoring(self, tmp_path: pathlib.Path) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream, load_path: typing.Optional[str] = None
        ) -> det.TrialController:
            return utils.make_trial_controller_from_trial_implementation(
                trial_class=la_model.OneVarTrial,
                hparams=self.hparams,
                workloads=workloads,
                load_path=load_path,
                trial_seed=self.trial_seed,
            )

        utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

    def test_checkpoint_save_load_hooks(self, tmp_path: pathlib.Path) -> None:
        def on_load_checkpoint(self, checkpoint: Dict[str, Any]):
            assert "test" in checkpoint
            assert checkpoint["test"] is True

        def on_save_checkpoint_true(self, checkpoint: Dict[str, Any], *args):
            checkpoint["test"] = True

        trial_cls_1 = fork_trial_override_lm(
            {
                "on_save_checkpoint": on_save_checkpoint_true,
                "on_load_checkpoint": on_load_checkpoint,
            }
        )

        def make_trial_controller_fn(
            workloads: workload.Stream, load_path: typing.Optional[str] = None
        ) -> det.TrialController:

            return utils.make_trial_controller_from_trial_implementation(
                trial_class=trial_cls_1,
                hparams=self.hparams,
                workloads=workloads,
                load_path=load_path,
                trial_seed=self.trial_seed,
            )

        utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

    def test_checkpoint_load_hook(self, tmp_path: pathlib.Path) -> None:
        def on_load_checkpoint(self, checkpoint: Dict[str, Any]):
            assert "test" in checkpoint

        trial_cls_1 = fork_trial_override_lm(
            {
                "on_load_checkpoint": on_load_checkpoint,
            }
        )

        def make_trial_controller_fn(
            workloads: workload.Stream, load_path: typing.Optional[str] = None
        ) -> det.TrialController:

            return utils.make_trial_controller_from_trial_implementation(
                trial_class=trial_cls_1,
                hparams=self.hparams,
                workloads=workloads,
                load_path=load_path,
                trial_seed=self.trial_seed,
            )

        with pytest.raises(AssertionError):
            utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

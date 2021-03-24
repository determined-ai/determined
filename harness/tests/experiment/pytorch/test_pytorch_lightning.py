import pathlib
import typing
from typing import Any, Dict

import pytest

import determined as det
from determined import pytorch, workload
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import lightning_adapter_onevar_model as la_model


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
        class OneVarLM(la_model.OneVarLM):
            def on_load_checkpoint(self, checkpoint: Dict[str, Any]):
                assert "test" in checkpoint
                assert checkpoint["test"] is True

            def on_save_checkpoint(self, checkpoint: Dict[str, Any]):
                checkpoint["test"] = True

        class OneVarLA(la_model.OneVarTrial):
            def __init__(self, context):
                super().__init__(context, OneVarLM)


        def make_trial_controller_fn(
            workloads: workload.Stream, load_path: typing.Optional[str] = None
        ) -> det.TrialController:

            return utils.make_trial_controller_from_trial_implementation(
                trial_class=OneVarLA,
                hparams=self.hparams,
                workloads=workloads,
                load_path=load_path,
                trial_seed=self.trial_seed,
            )

        utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

    def test_checkpoint_load_hook(self, tmp_path: pathlib.Path) -> None:
        class OneVarLM(la_model.OneVarLM):
            def on_load_checkpoint(self, checkpoint: Dict[str, Any]):
                assert "test" in checkpoint

        class OneVarLA(la_model.OneVarTrial):
            def __init__(self, context):
                super().__init__(context, OneVarLM)

        def make_trial_controller_fn(
            workloads: workload.Stream, load_path: typing.Optional[str] = None
        ) -> det.TrialController:

            return utils.make_trial_controller_from_trial_implementation(
                trial_class=OneVarLA,
                hparams=self.hparams,
                workloads=workloads,
                load_path=load_path,
                trial_seed=self.trial_seed,
            )

        with pytest.raises(AssertionError):
            utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

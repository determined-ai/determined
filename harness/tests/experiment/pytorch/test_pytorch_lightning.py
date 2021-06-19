import pathlib
import typing
from typing import Any, Dict

import pytest

import determined as det
from determined import workload
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import lightning_adapter_onevar_model as la_model


class TestLightningAdapter:
    def setup_method(self) -> None:
        # This training setup is not guaranteed to converge in general,
        # but has been tested with this random seed.  If changing this
        # random seed, verify the initial conditions converge.
        self.trial_seed = 17
        self.hparams = {
            "global_batch_size": 4,
        }

    def test_checkpointing_and_restoring(self, tmp_path: pathlib.Path) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream,
            checkpoint_dir: typing.Optional[str] = None,
            latest_checkpoint: typing.Optional[typing.Dict[str, typing.Any]] = None,
        ) -> det.TrialController:
            return utils.make_trial_controller_from_trial_implementation(
                trial_class=la_model.OneVarTrial,
                hparams=self.hparams,
                workloads=workloads,
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
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
            workloads: workload.Stream,
            checkpoint_dir: typing.Optional[str] = None,
            latest_checkpoint: typing.Optional[typing.Dict[str, typing.Any]] = None,
        ) -> det.TrialController:

            return utils.make_trial_controller_from_trial_implementation(
                trial_class=OneVarLA,
                hparams=self.hparams,
                workloads=workloads,
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
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
            workloads: workload.Stream,
            checkpoint_dir: typing.Optional[str] = None,
            latest_checkpoint: typing.Optional[typing.Dict[str, typing.Any]] = None,
        ) -> det.TrialController:

            return utils.make_trial_controller_from_trial_implementation(
                trial_class=OneVarLA,
                hparams=self.hparams,
                workloads=workloads,
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
            )

        with pytest.raises(AssertionError):
            utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

    def test_lr_scheduler(self, tmp_path: pathlib.Path) -> None:
        class OneVarLAFreq1(la_model.OneVarTrialLRScheduler):
            def check_lr_value(self, batch_idx: int):
                assert self.last_lr > self.read_lr_value()

        def make_trial_controller_fn(
            workloads: workload.Stream,
            checkpoint_dir: typing.Optional[str] = None,
            latest_checkpoint: typing.Optional[typing.Dict[str, typing.Any]] = None,
        ) -> det.TrialController:

            return utils.make_trial_controller_from_trial_implementation(
                trial_class=OneVarLAFreq1,
                hparams=self.hparams,
                workloads=workloads,
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
            )

        utils.train_and_validate(make_trial_controller_fn)

    def test_lr_scheduler_frequency(self) -> None:
        class OneVarLAFreq2(la_model.OneVarTrialLRScheduler):
            def check_lr_value(self, batch_idx: int):
                if batch_idx % 2 == 0:
                    assert self.last_lr > self.read_lr_value()
                else:
                    assert self.last_lr == self.read_lr_value()

        def make_trial_controller_fn(
            workloads: workload.Stream,
            checkpoint_dir: typing.Optional[str] = None,
            latest_checkpoint: typing.Optional[typing.Dict[str, typing.Any]] = None,
        ) -> det.TrialController:

            updated_params = {
                **self.hparams,
                "lr_frequency": 2,
            }
            return utils.make_trial_controller_from_trial_implementation(
                trial_class=OneVarLAFreq2,
                hparams=updated_params,
                workloads=workloads,
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
            )

        utils.train_and_validate(make_trial_controller_fn)

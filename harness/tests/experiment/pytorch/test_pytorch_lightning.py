# type: ignore
import pathlib
import random
import sys
import typing
from typing import Any, Dict

import pytest

import determined as det
from determined import gpu, pytorch
from determined.pytorch import lightning
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
        self.checkpoint_and_restore(
            trial_class=la_model.OneVarTrial, hparams=self.hparams, tmp_path=tmp_path, steps=(1, 1)
        )

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

        self.checkpoint_and_restore(
            trial_class=OneVarLA, hparams=self.hparams, tmp_path=tmp_path, steps=(1, 1)
        )

    def test_checkpoint_load_hook(self, tmp_path: pathlib.Path) -> None:
        class OneVarLM(la_model.OneVarLM):
            def on_load_checkpoint(self, checkpoint: Dict[str, Any]):
                assert "test" in checkpoint

        class OneVarLA(la_model.OneVarTrial):
            def __init__(self, context):
                super().__init__(context, OneVarLM)

        with pytest.raises(AssertionError):
            self.checkpoint_and_restore(
                trial_class=OneVarLA, hparams=self.hparams, tmp_path=tmp_path, steps=(1, 1)
            )

    def test_lr_scheduler(self, tmp_path: pathlib.Path) -> None:
        class OneVarLAFreq1(la_model.OneVarTrialLRScheduler):
            def check_lr_value(self, batch_idx: int):
                assert self.last_lr > self.read_lr_value()

        trial, trial_controller = create_trial_and_trial_controller(
            trial_class=OneVarLAFreq1,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=2,
            min_validation_batches=1,
            checkpoint_dir=str(tmp_path),
        )
        trial_controller.run()

    def test_lr_scheduler_frequency(self) -> None:
        class OneVarLAFreq2(la_model.OneVarTrialLRScheduler):
            def check_lr_value(self, batch_idx: int):
                if batch_idx % 2 == 0:
                    assert self.last_lr > self.read_lr_value()
                else:
                    assert self.last_lr == self.read_lr_value()

        updated_params = {
            **self.hparams,
            "lr_frequency": 2,
        }
        trial, trial_controller = create_trial_and_trial_controller(
            trial_class=OneVarLAFreq2,
            hparams=updated_params,
            trial_seed=self.trial_seed,
            max_batches=2,
            min_validation_batches=1,
        )
        trial_controller.run()

    def checkpoint_and_restore(
        self,
        hparams: typing.Dict,
        trial_class: pytorch.PyTorchTrial,
        tmp_path: pathlib.Path,
        steps: typing.Tuple[int, int] = (1, 1),
    ) -> typing.Tuple[
        typing.Sequence[typing.Dict[str, typing.Any]], typing.Sequence[typing.Dict[str, typing.Any]]
    ]:
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        training_metrics = {"A": [], "B": []}
        validation_metrics = {"A": [], "B": []}

        # Trial A: train 100 batches and checkpoint
        trial_A, trial_controller_A = create_trial_and_trial_controller(
            trial_class=trial_class,
            hparams=hparams,
            trial_seed=self.trial_seed,
            max_batches=steps[0],
            min_validation_batches=steps[0],
            min_checkpoint_batches=steps[0],
            checkpoint_dir=checkpoint_dir,
        )

        trial_controller_A.run()

        metrics_callback = trial_A.metrics_callback
        checkpoint_callback = trial_A.checkpoint_callback

        training_metrics["A"] = metrics_callback.training_metrics
        assert (
            len(training_metrics["A"]) == steps[0]
        ), "training metrics did not match expected length"
        validation_metrics["A"] = metrics_callback.validation_metrics

        assert len(checkpoint_callback.uuids) == 1, "trial did not return a checkpoint UUID"

        # Trial A: restore from checkpoint and train for 100 more batches
        trial_A, trial_controller_A = create_trial_and_trial_controller(
            trial_class=trial_class,
            hparams=hparams,
            trial_seed=self.trial_seed,
            max_batches=steps[0] + steps[1],
            min_validation_batches=steps[1],
            min_checkpoint_batches=sys.maxsize,
            checkpoint_dir=checkpoint_dir,
            latest_checkpoint=checkpoint_callback.uuids[0],
            steps_completed=trial_controller_A.state.batches_trained,
        )
        trial_controller_A.run()

        metrics_callback = trial_A.metrics_callback
        training_metrics["A"] += metrics_callback.training_metrics
        validation_metrics["A"] += metrics_callback.validation_metrics

        assert (
            len(training_metrics["A"]) == steps[0] + steps[1]
        ), "training metrics returned did not match expected length"

        # Trial B: run for 200 steps
        trial_B, trial_controller_B = create_trial_and_trial_controller(
            trial_class=trial_class,
            hparams=hparams,
            trial_seed=self.trial_seed,
            max_batches=steps[0] + steps[1],
            min_validation_batches=steps[0] + steps[1],
            min_checkpoint_batches=sys.maxsize,
            checkpoint_dir=checkpoint_dir,
        )
        trial_controller_B.run()

        metrics_callback = trial_B.metrics_callback

        training_metrics["B"] = metrics_callback.training_metrics
        validation_metrics["B"] = metrics_callback.validation_metrics

        for A, B in zip(training_metrics["A"], training_metrics["B"]):
            utils.assert_equivalent_metrics(A, B)

        for A, B in zip(validation_metrics["A"], validation_metrics["B"]):
            utils.assert_equivalent_metrics(A, B)

        return (training_metrics["A"], training_metrics["B"])


def create_trial_and_trial_controller(
    trial_class: lightning.LightningAdapter,
    hparams: typing.Dict,
    scheduling_unit: int = 1,
    trial_seed: int = None,
    exp_config: typing.Optional[typing.Dict] = None,
    checkpoint_dir: typing.Optional[str] = None,
    latest_checkpoint: typing.Optional[str] = None,
    steps_completed: int = 0,
    expose_gpus: bool = False,
    max_batches: int = 100,
    min_checkpoint_batches: int = sys.maxsize,
    min_validation_batches: int = sys.maxsize,
) -> typing.Tuple[pytorch.PyTorchTrial, pytorch._PyTorchTrialController]:
    assert issubclass(
        trial_class, pytorch.PyTorchTrial
    ), "pytorch test method called for non-pytorch trial"

    if not exp_config:
        assert hasattr(
            trial_class, "_searcher_metric"
        ), "Trial classes for unit tests should be annotated with a _searcher_metric attribute"
        searcher_metric = trial_class._searcher_metric
        exp_config = utils.make_default_exp_config(
            hparams, scheduling_unit, searcher_metric, checkpoint_dir=checkpoint_dir
        )

    if not trial_seed:
        trial_seed = random.randint(0, 1 << 31)

    storage_manager = det.common.storage.SharedFSStorageManager(checkpoint_dir or "/tmp")
    with det.core._dummy_init(storage_manager=storage_manager) as core_context:
        core_context.train._trial_id = "1"
        distributed_backend = det._DistributedBackend()
        if expose_gpus:
            gpu_uuids = gpu.get_gpu_uuids()
        else:
            gpu_uuids = []

        pytorch._PyTorchTrialController.pre_execute_hook(trial_seed, distributed_backend)
        trial_context = pytorch.PyTorchTrialContext(
            core_context=core_context,
            trial_seed=trial_seed,
            hparams=hparams,
            slots_per_trial=1,
            num_gpus=len(gpu_uuids),
            exp_conf=exp_config,
            aggregation_frequency=1,
            fp16_compression=False,
            average_aggregated_gradients=True,
            steps_completed=steps_completed,
            managed_training=False,
            debug_enabled=False,
        )

        trial_inst = trial_class(context=trial_context)

        trial_controller = pytorch._PyTorchTrialController(
            trial_inst=trial_inst,
            context=trial_context,
            max_length=pytorch.Batch(max_batches),
            checkpoint_period=pytorch.Batch(min_checkpoint_batches),
            validation_period=pytorch.Batch(min_validation_batches),
            searcher_metric_name=trial_class._searcher_metric,
            reporting_period=pytorch.Batch(scheduling_unit),
            local_training=True,
            latest_checkpoint=latest_checkpoint,
            steps_completed=steps_completed,
            smaller_is_better=bool(exp_config["searcher"]["smaller_is_better"]),
            test_mode=False,
            checkpoint_policy=exp_config["checkpoint_policy"],
            step_zero_validation=bool(exp_config["perform_initial_validation"]),
            det_profiler=None,
        )

        trial_controller._set_data_loaders()
        trial_controller.training_iterator = iter(trial_controller.training_loader)
        return trial_inst, trial_controller

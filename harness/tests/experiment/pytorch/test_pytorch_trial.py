# type: ignore
import contextlib
import importlib
import io
import os
import pathlib
import sys
import typing
from unittest import mock

import numpy as np
import pytest
import torch
from _pytest import monkeypatch
from torch.distributed import launcher

import determined as det
from determined import pytorch
from tests.experiment import pytorch_utils, utils  # noqa: I100
from tests.experiment.fixtures import pytorch_onevar_model

# Apex is included only for GPU trials.
try:
    import apex  # noqa

    HAVE_APEX = True
except ImportError:  # pragma: no cover
    HAVE_APEX = False
    pass


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


@pytest.mark.pytorch
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
            "dataloader_type": "determined",
            "disable_dataset_reproducibility_checks": False,
        }

    def test_onevar_single(self, tmp_path: pathlib.Path) -> None:
        """Assert that the training loss and validation error decrease monotonically."""
        tensorboard_path = tmp_path.joinpath("tensorboard")
        trial, trial_controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            tensorboard_path=tensorboard_path,
        )

        train_steps, metrics = trial_controller._train_with_boundaries(
            training_enumerator=enumerate(trial_controller.training_iterator),
            train_boundaries=[
                pytorch._TrainBoundary(
                    step_type=pytorch._TrainBoundaryType.TRAIN, unit=pytorch.Batch(100)
                )
            ],
        )

        assert len(train_steps) == 1, "unexpected train step count"
        assert train_steps[0].limit_reached, "train step did not reach expected limit"
        assert len(metrics) == 100, "metrics length did not match input"

        for i in range(100):
            pytorch_onevar_model.OneVarTrial.check_batch_metrics(
                metrics[i],
                i,
                metric_keyname_pairs=(("loss", "loss_exp"), ("w_after", "w_exp")),
            )
        for older, newer in zip(metrics, metrics[1:]):
            assert newer["loss"] <= older["loss"]

    def test_training_metrics(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        trial, trial_controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialWithTrainingMetrics,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            tensorboard_path=tensorboard_path,
        )

        train_steps, metrics = trial_controller._train_with_boundaries(
            training_enumerator=enumerate(trial_controller.training_iterator),
            train_boundaries=[
                pytorch._TrainBoundary(
                    step_type=pytorch._TrainBoundaryType.TRAIN, unit=pytorch.Batch(100)
                )
            ],
        )

        assert len(train_steps) == 1, "unexpected train step count"
        assert train_steps[0].limit_reached, "train step did not reach expected limit"
        assert len(metrics) == 100, "metrics length did not match input"
        for metric in metrics:
            assert "mse" in metric

    def test_nonscalar_validation(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        trial, trial_controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialWithNonScalarValidation,
            hparams=self.hparams,
            expose_gpus=True,
            trial_seed=self.trial_seed,
            tensorboard_path=tensorboard_path,
        )

        val_metrics = trial_controller._validate()

        assert "mse" in val_metrics

    def test_checkpointing_and_restoring(self, tmp_path: pathlib.Path) -> None:
        updated_hparams = {
            "lr_scheduler_step_mode": pytorch.LRScheduler.StepMode.STEP_EVERY_BATCH.value,
            **self.hparams,
        }
        self.checkpoint_and_check_metrics(
            pytorch_onevar_model.OneVarTrialWithLRScheduler, updated_hparams, tmp_path, (100, 100)
        )

    @pytest.mark.skipif(not torch.cuda.is_available(), reason="no gpu available")
    @pytest.mark.gpu
    @pytest.mark.parametrize(
        "trial_class",
        [
            pytorch_onevar_model.OneVarApexAMPTrial,
            pytorch_onevar_model.OneVarAutoAMPTrial,
            pytorch_onevar_model.OneVarManualAMPTrial,
        ],
        ids=[
            "apex",
            "autocast",
            "manual",
        ],
    )
    def test_scaler_checkpointing_and_restoring(self, trial_class, tmp_path: pathlib.Path) -> None:
        if trial_class is pytorch_onevar_model.OneVarApexAMPTrial and not HAVE_APEX:
            pytest.skip("Apex not available")

        updated_hparams = {
            "global_batch_size": 1,
            **self.hparams,
        }

        tm_a, tm_b = self.checkpoint_and_check_metrics(
            trial_class=pytorch_onevar_model.OneVarApexAMPTrial,
            hparams=updated_hparams,
            tmp_path=tmp_path,
            steps=(1, 1),
        )

        amp_metrics_test(trial_class, tm_a)
        amp_metrics_test(trial_class, tm_b)

    def test_restore_invalid_checkpoint(self, tmp_path: pathlib.Path) -> None:
        # Build, train, and save a checkpoint with the normal hyperparameters.
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        tensorboard_path = tmp_path.joinpath("tensorboard")

        # Trial A: run with 100 batches and checkpoint
        trial_A, trial_controller_A = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=100,
            min_checkpoint_batches=100,
            checkpoint_dir=checkpoint_dir,
            tensorboard_path=tensorboard_path,
        )

        trial_controller_A.run()

        assert len(trial_A.checkpoint_callback.uuids) == 1, "trial did not return a checkpoint UUID"

        # Trial A: restore from checkpoint with invalid hparams
        invalid_hparams = {**self.hparams, "features": 2}
        assert invalid_hparams != self.hparams

        with pytest.raises(RuntimeError):
            trial_A, trial_controller_A = pytorch_utils.create_trial_and_trial_controller(
                trial_class=pytorch_onevar_model.OneVarTrial,
                hparams=invalid_hparams,
                trial_seed=self.trial_seed,
                max_batches=100,
                min_validation_batches=100,
                min_checkpoint_batches=sys.maxsize,
                checkpoint_dir=checkpoint_dir,
                tensorboard_path=tensorboard_path,
                latest_checkpoint=trial_A.checkpoint_callback.uuids[0],
                steps_completed=trial_controller_A.state.batches_trained,
            )
            trial_controller_A.run()

    def test_reproducibility(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        training_metrics = {"A": [], "B": []}
        validation_metrics = {"A": [], "B": []}

        # Trial A
        trial_A, trial_controller_A = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=1000,
            min_validation_batches=100,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )
        trial_controller_A.run()

        metrics_callback = trial_A.metrics_callback
        training_metrics["A"] = metrics_callback.training_metrics
        validation_metrics["A"] = metrics_callback.validation_metrics

        # Trial B
        trial_B, trial_controller_B = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=1000,
            min_validation_batches=100,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )
        trial_controller_B.run()

        metrics_callback = trial_B.metrics_callback
        training_metrics["B"] = metrics_callback.training_metrics
        validation_metrics["B"] = metrics_callback.validation_metrics

        assert len(training_metrics["A"]) == len(training_metrics["B"])
        for A, B in zip(training_metrics["A"], training_metrics["B"]):
            utils.assert_equivalent_metrics(A, B)

        assert len(validation_metrics["A"]) == len(validation_metrics["B"])
        for A, B in zip(validation_metrics["A"], validation_metrics["B"]):
            utils.assert_equivalent_metrics(A, B)

    def test_custom_eval(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        training_metrics = {"A": [], "B": []}  # type: typing.Dict
        validation_metrics = {"A": [], "B": []}  # type: typing.Dict

        trial_A, trial_controller_A = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=900,
            min_validation_batches=100,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )
        trial_controller_A.run()

        metrics_callback = trial_A.metrics_callback

        training_metrics["A"] = metrics_callback.training_metrics
        validation_metrics["A"] = metrics_callback.validation_metrics

        trial_B, trial_controller_B = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialCustomEval,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            expose_gpus=True,
            max_batches=900,
            min_validation_batches=100,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )
        trial_controller_B.run()

        metrics_callback = trial_B.metrics_callback
        training_metrics["B"] = metrics_callback.training_metrics
        validation_metrics["B"] = metrics_callback.validation_metrics

        for original, custom_eval in zip(training_metrics["A"], training_metrics["B"]):
            assert np.allclose(original["loss"], custom_eval["loss"], atol=1e-6)

        for original, custom_eval in zip(validation_metrics["A"], validation_metrics["B"]):
            assert np.allclose(original["val_loss"], custom_eval["val_loss"], atol=1e-6)

    def test_grad_clipping(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        training_metrics = {"original": [], "clipped_by_norm": [], "clipped_by_val": []}
        validation_metrics = {"original": [], "clipped_by_norm": [], "clipped_by_val": []}

        trial, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialGradClipping,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )
        controller.run()

        metrics_callback = trial.metrics_callback

        training_metrics["original"] = metrics_callback.training_metrics
        validation_metrics["original"] = metrics_callback.validation_metrics

        updated_hparams = {"gradient_clipping_l2_norm": 0.0001, **self.hparams}
        trial, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialGradClipping,
            hparams=updated_hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )
        controller.run()

        metrics_callback = trial.metrics_callback

        training_metrics["clipped_by_norm"] = metrics_callback.training_metrics
        validation_metrics["clipped_by_norm"] = metrics_callback.validation_metrics

        for idx, (original, clipped) in enumerate(
            zip(training_metrics["original"], training_metrics["clipped_by_norm"])
        ):
            if idx < 10:
                continue
            assert original["loss"] != clipped["loss"]

        updated_hparams = {"gradient_clipping_value": 0.0001, **self.hparams}
        trial, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialGradClipping,
            hparams=updated_hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )
        controller.run()

        metrics_callback = trial.metrics_callback

        training_metrics["clipped_by_val"] = metrics_callback.training_metrics
        validation_metrics["clipped_by_val"] = metrics_callback.validation_metrics

        for idx, (original, clipped) in enumerate(
            zip(training_metrics["original"], training_metrics["clipped_by_val"])
        ):
            if idx < 10:
                continue
            assert original["loss"] != clipped["loss"]

    def test_per_metric_reducers(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        _, trial_controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialPerMetricReducers,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=2,
            min_validation_batches=1,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )
        trial_controller.run()

    def test_callbacks(self, tmp_path: pathlib.Path) -> None:
        checkpoint_dir = tmp_path.joinpath("checkpoint")
        tensorboard_path = tmp_path.joinpath("tensorboard")

        hparams1 = dict(self.hparams)
        hparams1["global_batch_size"] = 2
        training_epochs = 2
        num_batches = (
            training_epochs
            * len(pytorch_onevar_model.OnesDataset())
            // hparams1["global_batch_size"]
        )

        trial, trial_controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialCallbacks,
            hparams=hparams1,
            checkpoint_dir=str(checkpoint_dir),
            tensorboard_path=tensorboard_path,
            max_batches=num_batches,
            min_checkpoint_batches=sys.maxsize,
            min_validation_batches=sys.maxsize,
            scheduling_unit=sys.maxsize,
        )

        trial_controller.run()

        # Expect 2 training epochs, 1 validation step, and one checkpoint step
        assert trial.counter.__dict__ == {
            "trial_startups": 1,
            "validation_steps_started": 1,
            "validation_steps_ended": 1,
            "checkpoints_written": 1,
            "checkpoints_uploaded": 1,
            "training_started_times": 1,
            "training_epochs_started": 2,
            "training_epochs_ended": 2,
            "training_workloads_ended": 1,
            "trial_shutdowns": 1,
        }
        assert trial.legacy_counter.__dict__ == {"legacy_on_training_epochs_start_calls": 2}

        trial, trial_controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialCallbacks,
            hparams=hparams1,
            checkpoint_dir=str(checkpoint_dir),
            tensorboard_path=tensorboard_path,
            max_batches=num_batches,
            min_checkpoint_batches=sys.maxsize,
            min_validation_batches=num_batches // 2,
            scheduling_unit=sys.maxsize,
        )
        trial_controller.run()

        # Expect 1 training step,
        # 2 validation steps (1 specified + 1 from training finish),
        # 2 checkpoint steps (1 from validation + 1 from training finish)
        assert trial.counter.__dict__ == {
            "trial_startups": 1,
            "validation_steps_started": 2,
            "validation_steps_ended": 2,
            "checkpoints_written": 2,
            "checkpoints_uploaded": 2,
            "training_started_times": 1,
            "training_epochs_started": 2,
            "training_epochs_ended": 2,
            "training_workloads_ended": 2,
            "trial_shutdowns": 1,
        }
        assert trial.legacy_counter.__dict__ == {"legacy_on_training_epochs_start_calls": 2}

        trial, trial_controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialCallbacks,
            hparams=hparams1,
            checkpoint_dir=str(checkpoint_dir),
            tensorboard_path=tensorboard_path,
            max_batches=num_batches,
            min_checkpoint_batches=num_batches // 2,
            min_validation_batches=sys.maxsize,
            scheduling_unit=sys.maxsize,
        )
        trial_controller.run()

        # Expect 2 training steps (1 from checkpoint + 1 from training finish),
        # 1 validation steps (1 from training finish),
        # 2 checkpoint steps (1 from specified + 1 from training finish)
        assert trial.counter.__dict__ == {
            "trial_startups": 1,
            "validation_steps_started": 1,
            "validation_steps_ended": 1,
            "checkpoints_written": 2,
            "checkpoints_uploaded": 2,
            "training_started_times": 1,
            "training_epochs_started": 2,
            "training_epochs_ended": 2,
            "training_workloads_ended": 2,
            "trial_shutdowns": 1,
        }
        assert trial.legacy_counter.__dict__ == {"legacy_on_training_epochs_start_calls": 2}

    @pytest.mark.parametrize(
        "lr_scheduler_step_mode", [mode.value for mode in pytorch.LRScheduler.StepMode]
    )
    def test_context(
        self,
        tmp_path: pathlib.Path,
        lr_scheduler_step_mode,
    ) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        hparams = self.hparams.copy()
        hparams["lr_scheduler_step_mode"] = lr_scheduler_step_mode
        hparams["global_batch_size"] = 64

        _, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialAccessContext,
            hparams=hparams,
            trial_seed=self.trial_seed,
            max_batches=1,
            min_validation_batches=1,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )
        controller.run()

    def test_variable_workload_size(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        trial, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )

        training_metrics = []
        total_steps, total_batches_processed = 10, 0
        for step_id in range(1, total_steps):
            num_batches = step_id
            train_steps, metrics = controller._train_with_boundaries(
                training_enumerator=enumerate(controller.training_iterator),
                train_boundaries=[
                    pytorch._TrainBoundary(
                        step_type=pytorch._TrainBoundaryType.TRAIN, unit=pytorch.Batch(num_batches)
                    )
                ],
            )
            assert len(train_steps) == 1, "unexpected train step count"
            assert train_steps[0].limit_reached, "train step did not reach expected limit"
            assert len(metrics) == num_batches, "did not run for expected num_batches"
            training_metrics.extend(metrics)
            total_batches_processed += num_batches

        assert total_batches_processed == sum(
            range(1, total_steps)
        ), "total batches did not match expected"

    def test_custom_reducers(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        trial, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=30,
            min_validation_batches=30,
            min_checkpoint_batches=sys.maxsize,
            scheduling_unit=10,
            tensorboard_path=tensorboard_path,
        )
        controller.run()
        metrics_callback = trial.metrics_callback
        training_metrics = metrics_callback.training_metrics
        validation_metrics = metrics_callback.validation_metrics
        batch_size = self.hparams["global_batch_size"]

        for i, metrics in enumerate(training_metrics):
            expect = pytorch_onevar_model.TriangleLabelSum.expect(batch_size, 10 * i, 10 * (i + 1))
            assert "cls_reducer" in metrics
            assert metrics["cls_reducer"] == expect
            assert "fn_reducer" in metrics
            assert metrics["fn_reducer"] == expect

        for metrics in validation_metrics:
            num_batches = len(pytorch_onevar_model.OnesDataset()) // batch_size
            expect = pytorch_onevar_model.TriangleLabelSum.expect(batch_size, 0, num_batches)
            assert "cls_reducer" in metrics
            assert metrics["cls_reducer"] == expect
            assert "fn_reducer" in metrics
            assert metrics["fn_reducer"] == expect

    def test_reject_unnamed_nondict_metric(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        trial, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )

        def reducer_fn(_):
            return 1.0

        # Inject an unnamed metric which returns a non-dict (which is not allowed).
        controller.context.wrap_reducer(reducer_fn)

        with pytest.raises(AssertionError, match="name=None but it did not return a dict"):
            controller.run()

    def test_reject_named_dict_metric(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        # If at some point in the future the webui is able to render scalar metrics inside
        # nested dictionary metrics, this test could go away.

        _, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )

        def reducer_fn(_):
            return {"my_metric": 1.0}

        # Inject a named metric which returns a dict (which is not allowed).
        controller.context.wrap_reducer(reducer_fn, name="my_metric")

        with pytest.raises(AssertionError, match="with name set but it returned a dict anyway"):
            controller.run()

    def test_require_disable_dataset_reproducibility(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        hparams = dict(self.hparams)
        hparams["dataloader_type"] = "torch"
        hparams["disable_dataset_reproducibility_checks"] = False

        with pytest.raises(RuntimeError, match="you can disable this check by calling"):
            trial, controller = pytorch_utils.create_trial_and_trial_controller(
                trial_class=pytorch_onevar_model.OneVarTrial,
                hparams=hparams,
                trial_seed=self.trial_seed,
                max_batches=100,
                min_validation_batches=10,
                min_checkpoint_batches=sys.maxsize,
                tensorboard_path=tensorboard_path,
            )
            controller.run()

    def test_custom_dataloader(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        hparams = dict(self.hparams)
        hparams["dataloader_type"] = "torch"
        hparams["disable_dataset_reproducibility_checks"] = True

        trial, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )
        controller.run()

        metrics_callback = trial.metrics_callback
        training_metrics = metrics_callback.training_metrics

        # Check the gradient update at every step.
        for idx, batch_metrics in enumerate(training_metrics):
            pytorch_onevar_model.OneVarTrial.check_batch_metrics(
                batch_metrics,
                idx,
                metric_keyname_pairs=(("loss", "loss_exp"), ("w_after", "w_exp")),
            )

        # We expect the validation error and training loss to be
        # monotonically decreasing.
        for older, newer in zip(training_metrics, training_metrics[1:]):
            assert newer["loss"] <= older["loss"]

    def test_gradient_aggregation(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        AGG_FREQ = 2
        exp_config = utils.make_default_exp_config(
            self.hparams,
            scheduling_unit=1,
            searcher_metric=pytorch_onevar_model.OneVarTrial._searcher_metric,
        )
        exp_config["optimizations"].update(
            {
                "aggregation_frequency": AGG_FREQ,
                "average_aggregated_gradients": True,
            }
        )

        trial, controller = pytorch_utils.create_trial_and_trial_controller(
            exp_config=exp_config,
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )

        controller.run()

        metrics_callback = trial.metrics_callback
        training_metrics = metrics_callback.training_metrics
        # Check the gradient update at every step.
        for idx, batch_metrics in enumerate(training_metrics):
            if (idx + 1) % AGG_FREQ != 0:
                # Only test batches which land on aggregation_frequency boundaries.
                continue
            pytorch_onevar_model.OneVarTrial.check_batch_metrics(
                batch_metrics,
                idx,
                metric_keyname_pairs=(("loss", "loss_exp"), ("w_after", "w_exp")),
            )

        for older, newer in zip(training_metrics, training_metrics[1:]):
            assert newer["loss"] <= older["loss"]

    @pytest.mark.skipif(not torch.cuda.is_available(), reason="no gpu available")
    @pytest.mark.gpu
    @pytest.mark.parametrize(
        "trial_class",
        [
            pytorch_onevar_model.OneVarApexAMPTrial,
            pytorch_onevar_model.OneVarAutoAMPTrial,
            pytorch_onevar_model.OneVarManualAMPTrial,
            pytorch_onevar_model.OneVarManualAMPWithNoopApexTrial,
            pytorch_onevar_model.OneVarApexAMPWithNoopScalerTrial,
        ],
        ids=[
            "apex",
            "autocast",
            "manual",
            "manual-with-noop-apex",
            "apex-with-noop-scaler",
        ],
    )
    def test_amp(self, tmp_path: pathlib.Path, trial_class) -> None:
        """Train a linear model using Determined with Automated Mixed Precision in three ways:
        Using Apex and using PyTorch AMP both "automatically" and "manually". In the "manual" case,
        we use the context manager ``autoscale`` in the model's training and
        evaluating methods; a scaler object is wrapped in a Determined context. The same
        is done under the hood in the first two cases.
        """
        tensorboard_path = tmp_path.joinpath("tensorboard")

        if trial_class is pytorch_onevar_model.OneVarApexAMPTrial and not HAVE_APEX:
            pytest.skip("Apex not available")

        # The assertions logic in make_amp_workloads require a batch size of one
        hparams = dict(self.hparams)
        hparams["global_batch_size"] = 1

        trial, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=trial_class,
            hparams=hparams,
            trial_seed=self.trial_seed,
            expose_gpus=True,
            max_batches=20,
            min_validation_batches=1,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
        )

        controller.run()

        metrics_callback = trial.metrics_callback
        training_metrics = metrics_callback.training_metrics

        amp_metrics_test(trial_class, training_metrics)

    @pytest.mark.skipif(not torch.cuda.is_available(), reason="no gpu available")
    @pytest.mark.gpu
    @pytest.mark.parametrize(
        "trial_class",
        [
            pytorch_onevar_model.OneVarApexAMPTrial,
            pytorch_onevar_model.OneVarAutoAMPTrial,
            pytorch_onevar_model.OneVarManualAMPTrial,
        ],
        ids=[
            "apex",
            "autocast",
            "manual",
        ],
    )
    def test_amp_with_gradient_aggregation(self, tmp_path: pathlib.Path, trial_class) -> None:
        """Similar to test_amp but with gradient aggregation."""
        tensorboard_path = tmp_path.joinpath("tensorboard")

        if trial_class is pytorch_onevar_model.OneVarApexAMPTrial and not HAVE_APEX:
            pytest.skip("Apex not available")

        # The assertions logic in make_amp_workloads require a batch size of one
        hparams = dict(self.hparams)
        hparams["global_batch_size"] = 1
        aggregation_frequency = 2

        exp_config = utils.make_default_exp_config(
            hparams,
            scheduling_unit=1,
            searcher_metric=trial_class._searcher_metric,
        )

        trial, trial_controller = pytorch_utils.create_trial_and_trial_controller(
            exp_config=exp_config,
            trial_class=trial_class,
            hparams=hparams,
            trial_seed=self.trial_seed,
            expose_gpus=True,
            max_batches=20 * aggregation_frequency,
            min_validation_batches=1,
            min_checkpoint_batches=sys.maxsize,
            tensorboard_path=tensorboard_path,
            aggregation_frequency=aggregation_frequency,
        )
        trial_controller.run()

        metrics_callback = trial.metrics_callback
        training_metrics = metrics_callback.training_metrics

        amp_metrics_test(trial_class, training_metrics, agg_freq=aggregation_frequency)

    def test_disable_tb_logging(self, tmp_path: pathlib.Path) -> None:
        tensorboard_path = tmp_path.joinpath("tensorboard")

        trial, trial_controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            tensorboard_path=tensorboard_path,
        )
        trial.context.set_enable_tensorboard_logging(False)
        trial_controller._upload_tb_files = mock.MagicMock()
        trial_controller.run()

        assert trial_controller._upload_tb_files.call_count == 0

    def test_trainer(self, monkeypatch: monkeypatch.MonkeyPatch, tmp_path: pathlib.Path) -> None:
        # there is no direct way to set tensorboard path in Trainer API
        def mock_get_tensorboard_path(dummy: typing.Dict[str, typing.Any]) -> pathlib.Path:
            return tmp_path.joinpath("tensorboard")

        monkeypatch.setattr(
            pytorch.PyTorchTrialContext, "get_tensorboard_path", mock_get_tensorboard_path
        )

        # Train for 100 batches, checkpoint and validate every 50 batches
        max_batches = 100
        with pytorch.init(hparams=self.hparams) as train_context:
            trial = pytorch_onevar_model.OneVarTrial(train_context)
            trainer = pytorch.Trainer(trial, train_context)
            trainer.fit(
                max_length=pytorch.Batch(max_batches),
                checkpoint_period=pytorch.Batch(max_batches // 2),
                validation_period=pytorch.Batch(max_batches // 2),
            )

        # Verify training and validation metrics for batches trained
        metrics_callback = trial.metrics_callback
        batch_metrics = metrics_callback.batch_metrics
        assert len(batch_metrics) == max_batches, "batch metrics did not match expected length"

        validation_metrics = metrics_callback.validation_metrics
        assert len(validation_metrics) == 2, "validation metrics did not match expected length"

        # Verify checkpoint
        checkpoint_callback = trial.checkpoint_callback
        assert (
            len(checkpoint_callback.uuids) == 2
        ), "checkpoint callback did not return expected length of uuids"

    def test_trainer_callbacks(
        self, monkeypatch: monkeypatch.MonkeyPatch, tmp_path: pathlib.Path
    ) -> None:
        # there is no direct way to set tensorboard path in Trainer API
        def mock_get_tensorboard_path(dummy: typing.Dict[str, typing.Any]) -> pathlib.Path:
            return tmp_path.joinpath("tensorboard")

        monkeypatch.setattr(
            pytorch.PyTorchTrialContext, "get_tensorboard_path", mock_get_tensorboard_path
        )

        max_epochs = 2
        checkpoint_batches = 5
        validation_batches = 10

        with pytorch.init(hparams=self.hparams) as train_context:
            trial = pytorch_onevar_model.OneVarTrialCallbacks(train_context)
            trainer = pytorch.Trainer(trial, train_context)
            trainer.fit(
                max_length=pytorch.Epoch(2),
                checkpoint_period=pytorch.Batch(checkpoint_batches),
                validation_period=pytorch.Batch(validation_batches),
            )

        # Expect epochs * epoch_len / period if last batch is end of epoch,
        # else + 1 for last checkpoint/validation
        total_batches = max_epochs * train_context._epoch_len
        checkpoint_periods, batches_remaining = divmod(total_batches, checkpoint_batches)
        checkpoints = checkpoint_periods + 1 if batches_remaining > 0 else checkpoint_periods
        validation_periods, batches_remaining = divmod(total_batches, validation_batches)
        validations = validation_periods + 1 if batches_remaining > 0 else validation_periods

        workload_steps = max(checkpoints, validations)

        assert trial.counter.__dict__ == {
            "trial_startups": 1,
            "validation_steps_started": validations,
            "validation_steps_ended": validations,
            "checkpoints_written": checkpoints,
            "checkpoints_uploaded": checkpoints,
            "training_started_times": 1,
            "training_epochs_started": max_epochs,
            "training_epochs_ended": max_epochs,
            "training_workloads_ended": workload_steps,
            "trial_shutdowns": 1,
        }

        assert trial.legacy_counter.__dict__ == {"legacy_on_training_epochs_start_calls": 2}

    @pytest.mark.parametrize(
        "enable_tensorboard_logging",
        [True, False],
        ids=["tensorboard logging enabled", "tensorboard logging disabled"],
    )
    def test_trainer_disable_tb_logging(
        self,
        monkeypatch: monkeypatch.MonkeyPatch,
        tmp_path: pathlib.Path,
        enable_tensorboard_logging: bool,
    ):
        # there is no direct way to set tensorboard path in Trainer API
        def mock_get_tensorboard_path(dummy: typing.Dict[str, typing.Any]) -> pathlib.Path:
            return tmp_path.joinpath("tensorboard")

        monkeypatch.setattr(
            pytorch.PyTorchTrialContext, "get_tensorboard_path", mock_get_tensorboard_path
        )

        checkpoint_batches = 5
        validation_batches = 5

        with mock.patch.object(det.pytorch, "_log_tb_metrics", return_value=None) as mock_method:
            with pytorch.init(
                hparams=self.hparams, enable_tensorboard_logging=enable_tensorboard_logging
            ) as train_context:
                trial = pytorch_onevar_model.OneVarTrialCallbacks(train_context)
                trainer = pytorch.Trainer(trial, train_context)
                trainer.fit(
                    max_length=pytorch.Epoch(1),
                    checkpoint_period=pytorch.Batch(checkpoint_batches),
                    validation_period=pytorch.Batch(validation_batches),
                )
            if enable_tensorboard_logging:
                assert mock_method.call_count > 0
            else:
                assert mock_method.call_count == 0

    @pytest.mark.skipif(torch.cuda.device_count() < 2, reason="not enough gpus")
    @pytest.mark.gpu_parallel
    def test_gradient_aggregation_parallel(self, tmp_path: pathlib.Path):
        launch_config = pytorch_utils.setup_torch_distributed()

        val_metrics = launcher.elastic_launch(launch_config, run_identity)(tmp_path)

        # weights returned by both models are the same.
        model_1_metrics = val_metrics[0]
        model_1_weights = [model_1_metrics[i]["weight"] for i in range(len(model_1_metrics))]
        model_2_metrics = val_metrics[1]
        model_2_weights = [model_2_metrics[i]["weight"] for i in range(len(model_2_metrics))]

        expected_weights = calculate_gradients(num_epochs=1)

        assert model_1_weights == pytest.approx(
            expected_weights
        ), f"{model_1_weights} != {expected_weights}"

        assert model_2_weights == pytest.approx(
            expected_weights
        ), f"{model_2_weights} != {expected_weights}"

    @pytest.mark.skipif(torch.cuda.device_count() < 2, reason="not enough gpus")
    @pytest.mark.gpu_parallel
    @pytest.mark.parametrize("api_style", ["apex", "auto", "manual"])
    def test_pytorch_distributed_with_amp(self, tmp_path: pathlib.Path, api_style):
        launch_config = pytorch_utils.setup_torch_distributed()

        outputs = launcher.elastic_launch(launch_config, run_amp)(tmp_path, api_style)
        launcher.elastic_launch(launch_config, run_amp)(tmp_path, api_style, outputs[0])

    @pytest.mark.skipif(torch.cuda.device_count() < 2, reason="not enough gpus")
    @pytest.mark.gpu_parallel
    def test_distributed_logging(self, tmp_path: pathlib.Path):
        num_procs = 2

        launch_config = pytorch_utils.setup_torch_distributed(local_procs=num_procs)

        outputs = launcher.elastic_launch(launch_config, run_no_op)(tmp_path)

        log_output = sum([outputs[i] for i in range(num_procs)], [])

        patterns = [f"finished train_batch for rank {i}" for i in range(num_procs)]

        utils.assert_patterns_in_logs(log_output, patterns)

    @pytest.mark.skipif(torch.cuda.device_count() < 2, reason="not enough gpus")
    @pytest.mark.gpu_parallel
    @pytest.mark.parametrize("dataset_len", [2, 3])
    def test_epoch_sync(self, tmp_path: pathlib.Path, dataset_len: int):
        num_procs = 2

        launch_config = pytorch_utils.setup_torch_distributed(local_procs=num_procs)

        num_steps = 10
        global_batch_size = 2
        outputs = launcher.elastic_launch(launch_config, run_no_op)(
            tmp_path, num_steps, global_batch_size, dataset_len
        )

        log_output = sum([outputs[i] for i in range(num_procs)], [])

        batches_per_epoch = (dataset_len + global_batch_size - 1) // global_batch_size  # ceil

        patterns = []
        for rank in range(num_procs):
            for batch_idx in range(num_steps):
                epoch_idx = batch_idx // batches_per_epoch
                patterns.append(f"rank {rank} finished batch {batch_idx} in epoch {epoch_idx}")

        utils.assert_patterns_in_logs(log_output, patterns)

    @pytest.mark.parametrize(
        "max_batches,steps_completed",
        [
            (5, 5),
            (5, 10),
            (6, 10),
        ],
    )
    def test_max_batches_leq_steps_completed(
        self, max_batches: int, steps_completed: int, tmp_path: pathlib.Path
    ):
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        trial_A, trial_controller_A = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            checkpoint_dir=checkpoint_dir,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=steps_completed,
            min_validation_batches=steps_completed,
            min_checkpoint_batches=steps_completed,
        )
        trial_controller_A.run()

        checkpoint_callback = trial_A.checkpoint_callback
        assert len(checkpoint_callback.uuids) == 1, "trial did not return a checkpoint UUID"

        trial_B, trial_controller_B = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            checkpoint_dir=checkpoint_dir,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=max_batches,
            min_validation_batches=1,
            min_checkpoint_batches=1,
            latest_checkpoint=checkpoint_callback.uuids[0],
        )
        trial_controller_B.run()

        assert len(trial_B.metrics_callback.validation_metrics) == 0
        assert len(trial_B.metrics_callback.training_metrics) == 0

    def checkpoint_and_check_metrics(
        self,
        trial_class: pytorch_onevar_model.OneVarTrial,
        hparams: typing.Dict,
        tmp_path: pathlib.Path,
        steps: typing.Tuple[int, int] = (1, 1),
    ) -> typing.Tuple[
        typing.Sequence[typing.Dict[str, typing.Any]], typing.Sequence[typing.Dict[str, typing.Any]]
    ]:
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        tensorboard_path = tmp_path.joinpath("tensorboard")
        training_metrics = {"A": [], "B": []}
        validation_metrics = {"A": [], "B": []}

        # Trial A: train some batches and checkpoint
        trial_A, trial_controller_A = pytorch_utils.create_trial_and_trial_controller(
            trial_class=trial_class,
            hparams=hparams,
            trial_seed=self.trial_seed,
            max_batches=steps[0],
            min_validation_batches=steps[0],
            min_checkpoint_batches=steps[0],
            checkpoint_dir=checkpoint_dir,
            tensorboard_path=tensorboard_path,
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

        # Trial A: restore from checkpoint and train
        trial_A, trial_controller_A = pytorch_utils.create_trial_and_trial_controller(
            trial_class=trial_class,
            hparams=hparams,
            trial_seed=self.trial_seed,
            max_batches=steps[0] + steps[1],
            min_validation_batches=steps[1],
            min_checkpoint_batches=sys.maxsize,
            checkpoint_dir=checkpoint_dir,
            tensorboard_path=tensorboard_path,
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

        # Trial B: run for some steps
        trial_B, trial_controller_B = pytorch_utils.create_trial_and_trial_controller(
            trial_class=trial_class,
            hparams=hparams,
            trial_seed=self.trial_seed,
            max_batches=steps[0] + steps[1],
            min_validation_batches=steps[0],
            min_checkpoint_batches=sys.maxsize,
            checkpoint_dir=checkpoint_dir,
            tensorboard_path=tensorboard_path,
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

    def test_trial_validation_checkpointing(self, tmp_path: pathlib.Path):
        tensorboard_path = tmp_path.joinpath("tensorboard")

        trial, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            tensorboard_path=tensorboard_path,
        )

        # Checkpoint only if the following conditions are met:
        # - the checkpoint is not current
        # - the best validation metric returned is better per smaller_is_better
        checkpoint_conditions = [
            {
                "checkpoint_is_current": False,
                "best_validation": float("inf"),
                "smaller_is_better": True,
                "checkpoint": True,
            },
            {
                "checkpoint_is_current": False,
                "best_validation": float("-inf"),
                "smaller_is_better": False,
                "checkpoint": True,
            },
            {
                "checkpoint_is_current": False,
                "best_validation": sys.maxsize,
                "smaller_is_better": False,
                "checkpoint": False,
            },
            {
                "checkpoint_is_current": True,
                "best_validation": sys.maxsize,
                "smaller_is_better": False,
                "checkpoint": False,
            },
        ]
        for checkpoint_condition in checkpoint_conditions:
            controller.smaller_is_better = checkpoint_condition["smaller_is_better"]
            controller._checkpoint_is_current = mock.MagicMock(
                return_value=checkpoint_condition["checkpoint_is_current"]
            )
            controller.core_context.train.get_experiment_best_validation = mock.MagicMock(
                return_value=checkpoint_condition["best_validation"]
            )
            controller._checkpoint = mock.MagicMock()
            controller._validate(det.core.DummySearcherOperation(length=100, is_chief=True))
            controller.core_context.train.get_experiment_best_validation.assert_called_once()
            if checkpoint_condition["checkpoint"]:
                controller._checkpoint.assert_called_once()
            controller.core_context.train.get_experiment_best_validation.reset_mock()
            controller._checkpoint.reset_mock()

    def test_searcher_progress_reporting(self):
        trial, controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            scheduling_unit=10,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
        )

        controller._report_searcher_progress = mock.MagicMock()

        controller.run()

        # Expect progress reports every scheduling unit step + 1 on training end.
        assert controller._report_searcher_progress.call_count == (100 / 10) + 1

    @pytest.mark.parametrize(
        "ckpt",
        [
            "0.20.0-pytorch",
        ],
    )
    def test_legacy_checkpoint_loading(self, tmp_path: pathlib.Path, ckpt: str):
        """
        This test exists to validate the checkpoint load path from older checkpoints into
        post-Trainer API checkpoints. Trainer API deprecated workload_sequencer.pkl and
        replaced it with trial_state.pkl. It can be deleted some time after Trainer API release.
        """
        checkpoint_dir = os.path.join(utils.fixtures_path("ancient-checkpoints"), f"{ckpt}")
        tensorboard_path = tmp_path.joinpath("tensorboard")

        trial, trial_controller = pytorch_utils.create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams={"dataloader_type": "determined", "global_batch_size": 16},
            trial_seed=0,
            max_batches=1,
            min_validation_batches=1,
            min_checkpoint_batches=1,
            checkpoint_dir=checkpoint_dir,
            tensorboard_path=tensorboard_path,
        )

        # Manually set trial ID to match checkpoint.
        trial_controller.trial_id = 1

        # Load checkpoint.
        trial_controller._load(pathlib.Path(checkpoint_dir))

        # Verify checkpoint loaded state.
        state = trial_controller.state
        assert state.trial_id == 1, "trial_id does not match"
        assert state.last_ckpt == 1, "last_ckpt does not match"
        assert state.step_id == 1, "step_id does not match"
        assert state.last_val == 0, "last_val does not match"
        assert state.batches_trained == 1, "batches_trained does not match"
        assert state.epochs_trained == 0, "epochs_trained does not match"

    @pytest.mark.gpu
    @pytest.mark.cpu
    def test_rng_restore(self, tmp_path: pathlib.Path):
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        tensorboard_path = tmp_path.joinpath("tensorboard")

        config_base = utils.load_config(utils.fixtures_path("pytorch_no_op/const.yaml"))
        hparams = config_base["hyperparameters"]

        exp_config = utils.make_default_exp_config(
            hparams,
            scheduling_unit=1,
            searcher_metric="validation_loss",
            checkpoint_dir=checkpoint_dir,
        )
        exp_config.update(config_base)

        example_path = utils.fixtures_path("pytorch_no_op/model_def.py")
        trial_class = utils.import_class_from_module("NoopPyTorchTrial", example_path)
        trial_class._searcher_metric = "validation_error"

        trial_A, trial_controller_A = pytorch_utils.create_trial_and_trial_controller(
            trial_class=trial_class,
            hparams=hparams,
            trial_seed=self.trial_seed,
            exp_config=exp_config,
            max_batches=5,
            min_validation_batches=1,
            min_checkpoint_batches=1,
            checkpoint_dir=checkpoint_dir,
            tensorboard_path=tensorboard_path,
            expose_gpus=True,
        )

        trial_controller_A.run()

        # reset random seed before rerun
        trial_controller_A._set_random_seeds(0)

        checkpoints = trial_A.checkpoint_callback.uuids

        assert len(checkpoints) == 5, "trial did not create all checkpoints"

        # Trial B: restore from checkpoint and train for 4 more batches, not passing trial seed
        trial_B, trial_controller_B = pytorch_utils.create_trial_and_trial_controller(
            trial_class=trial_class,
            hparams=hparams,
            exp_config=exp_config,
            max_batches=5,
            min_validation_batches=1,
            min_checkpoint_batches=1,
            checkpoint_dir=checkpoint_dir,
            tensorboard_path=tensorboard_path,
            latest_checkpoint=checkpoints[0],
            steps_completed=1,
            expose_gpus=True,
        )
        trial_controller_B.run()

        # compare every aligning batch
        metrics_before = trial_A.metrics_callback.validation_metrics[1:]
        metrics_after = trial_B.metrics_callback.validation_metrics

        assert metrics_before == metrics_after, "mismatched metrics in RNG restore"


@pytest.mark.pytorch
@pytest.mark.parametrize(
    "ckpt,istrial,trial_spec,trial_kwargs",
    [
        ("0.13.13-pytorch-old", False, None, {}),
        ("0.13.13-pytorch-flex", True, None, {}),
        ("0.17.6-pytorch", True, None, {}),
        ("0.17.7-pytorch", True, None, {}),
        ("0.20.0-pytorch", True, None, {}),
        ("0.21.0-pytorch", True, None, {"lr": 0.001}),
        ("0.21.0-pytorch-main", True, "train:OneVarPytorchTrial", {"lr": 0.001}),
        # Test overriding the class even when the class is auto-importable:
        ("0.21.0-pytorch", True, "model_def:OneVarPytorchTrial", {"lr": 0.001}),
    ],
)
def test_trial_checkpoint_loading(
    ckpt: str,
    istrial: bool,
    trial_spec: typing.Optional[str],
    trial_kwargs: typing.Dict[str, typing.Any],
):
    checkpoint_dir = os.path.join(utils.fixtures_path("ancient-checkpoints"), f"{ckpt}")
    trial_class = None
    if trial_spec:
        with det.import_from_path(os.path.join(checkpoint_dir, "code")):
            file, cls = trial_spec.split(":")
            module = importlib.import_module(file)
            trial_class = getattr(module, cls)
    trial = pytorch.load_trial_from_checkpoint_path(
        checkpoint_dir,
        trial_class=trial_class,
        trial_kwargs=trial_kwargs,
        torch_load_kwargs={"map_location": "cpu"},
    )
    if istrial:
        assert isinstance(trial, pytorch.PyTorchTrial), type(trial)
    else:
        assert isinstance(trial, torch.nn.Module), type(trial)


def amp_metrics_test(trial_class, training_metrics, agg_freq=1):
    loss_prev = None
    GROWTH_INTERVAL = trial_class._growth_interval
    MIN_SCALED_LOSS_TO_REDUCE_SCALE = 32760
    growth_countdown = GROWTH_INTERVAL
    # Only attempt assertions up to and including the penultimate batch, because
    #  we may not have the updated scale from the final batch.
    for idx, (metrics, next_metrics) in enumerate(zip(training_metrics[:-1], training_metrics[1:])):
        if (idx + 1) % agg_freq != 0:
            # Only test batches which land on aggregation_frequency boundaries.
            continue
        # Because the scaler is updated during the optimizer step, which occurs after training from
        #  a batch, the metrics dictionary may not have the updated scale, but we can get it
        #  from the next step.
        scale = next_metrics["scale_before"]
        if "scale" in metrics:
            # In cases where we do know the scale immediately after it's updated, we might as well
            #  do this check. If this fails, something is very wrong.
            assert metrics["scale"] == scale, "scale is inconsistent between batches"
        else:
            metrics["scale"] = scale
        loss = metrics["loss"]
        scale_before = metrics["scale_before"]
        scaled_loss = loss * scale_before
        scale = metrics["scale"]
        growth_countdown -= 1
        if scaled_loss >= MIN_SCALED_LOSS_TO_REDUCE_SCALE:
            assert scale < scale_before, (
                f"scale was expected to reduce from {scale_before} but did not "
                f"(scaled_loss={scaled_loss} >= {MIN_SCALED_LOSS_TO_REDUCE_SCALE})) "
            )
            growth_countdown = GROWTH_INTERVAL
        elif growth_countdown == 0:
            assert scale > scale_before, (
                f"scale was expected to grow but did not " f"(growth_countdown={growth_countdown}) "
            )
            growth_countdown = GROWTH_INTERVAL
        else:
            assert scale == scale_before, (
                f"scale changed from {scale_before} to {scale} but not when expected "
                f"(growth_countdown={growth_countdown}) "
                f"(scaled_loss={scaled_loss} < {MIN_SCALED_LOSS_TO_REDUCE_SCALE})) "
            )
            # Check the accuracy of the gradient change.
            metric_keyname_pairs = [("loss", "loss_exp"), ("w_after", "w_exp")]
            if metrics["stage"] in ["small", "zero"]:
                metric_keyname_pairs.append(("w_before", "w_after"))
            trial_class.check_batch_metrics(
                metrics,
                idx,
                metric_keyname_pairs=metric_keyname_pairs,
                atol=1e-4,
            )
            if loss_prev is not None and metrics["stage"] == "one":
                assert loss <= loss_prev, "loss was expected to decrease monotonically"
                loss_prev = loss


def run_identity(tmp_path: pathlib.Path):
    checkpoint_dir = str(tmp_path.joinpath("checkpoint"))

    config = utils.load_config(utils.fixtures_path("pytorch_identity/distributed.yaml"))
    hparams = config["hyperparameters"]

    exp_config = utils.make_default_exp_config(
        hparams,
        scheduling_unit=1,
        searcher_metric="validation_loss",
        checkpoint_dir=checkpoint_dir,
    )
    exp_config.update(config)
    exp_config["searcher"]["smaller_is_better"] = True

    # each subprocess must import separately as trial_class cannot be pickled.
    example_path = utils.fixtures_path("pytorch_identity/model_def.py")
    trial_class = utils.import_class_from_module("IdentityPyTorchTrial", example_path)
    trial_class._searcher_metric = "weight"

    tensorboard_path = tmp_path.joinpath("tensorboard")

    trial, trial_controller = pytorch_utils.create_trial_and_trial_controller(
        trial_class=trial_class,
        hparams=hparams,
        slots_per_trial=2,
        max_batches=16,
        min_validation_batches=1,
        min_checkpoint_batches=16,
        checkpoint_dir=checkpoint_dir,
        tensorboard_path=tensorboard_path,
        aggregation_frequency=2,
    )

    trial_controller.run()

    metrics_callback = trial.metrics_callback

    validation_metrics = metrics_callback.validation_metrics

    return validation_metrics


def run_amp(tmp_path: pathlib.Path, api_style: str, batches_trained: typing.Optional[int] = 0):
    checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
    class_selector = {
        "apex": "MNistApexAMPTrial",
        "auto": "MNistAutoAMPTrial",
        "manual": "MNistManualAMPTrial",
    }

    config = utils.load_config(utils.fixtures_path(f"pytorch_amp/{api_style}_amp_distributed.yaml"))
    config = config.copy()
    config.setdefault("profiling", {})
    config["profiling"]["enabled"] = True

    hparams = config["hyperparameters"]

    exp_config = utils.make_default_exp_config(
        hparams,
        scheduling_unit=1,
        searcher_metric="validation_loss",
        checkpoint_dir=checkpoint_dir,
    )
    exp_config.update(config)
    exp_config["searcher"]["smaller_is_better"] = True

    example_path = utils.fixtures_path(f"pytorch_amp/{api_style}_amp_model_def.py")
    trial_class = utils.import_class_from_module(class_selector[api_style], example_path)
    trial_class._searcher_metric = "validation_loss"

    if batches_trained == 0:
        return pytorch_utils.train_for_checkpoint(
            trial_class=trial_class,
            hparams=hparams,
            slots_per_trial=2,
            tmp_path=tmp_path,
            exp_config=exp_config,
            steps=1,
        )
    else:
        pytorch_utils.train_from_checkpoint(
            trial_class=trial_class,
            hparams=hparams,
            slots_per_trial=2,
            tmp_path=tmp_path,
            exp_config=exp_config,
            steps=(1, 1),
            batches_trained=batches_trained,
        )
        return True


def run_no_op(
    tmp_path: pathlib.Path,
    num_steps: int = 1,
    global_batch_size: int = 32,
    dataset_len: int = 64,
):
    checkpoint_dir = str(tmp_path.joinpath("checkpoint"))

    config = utils.load_config(utils.fixtures_path("pytorch_no_op/const.yaml"))
    hparams = config["hyperparameters"]
    hparams["dataset_len"] = dataset_len
    hparams["global_batch_size"] = global_batch_size

    exp_config = utils.make_default_exp_config(
        hparams,
        scheduling_unit=1,
        searcher_metric="validation_loss",
        checkpoint_dir=checkpoint_dir,
    )
    exp_config.update(config)
    exp_config["searcher"]["smaller_is_better"] = True

    example_path = utils.fixtures_path("pytorch_no_op/model_def.py")
    trial_class = utils.import_class_from_module("NoopPyTorchTrial", example_path)
    trial_class._searcher_metric = "validation_error"

    f = io.StringIO()

    with contextlib.redirect_stdout(f):
        pytorch_utils.train_for_checkpoint(
            hparams=hparams,
            trial_class=trial_class,
            tmp_path=tmp_path,
            exp_config=exp_config,
            slots_per_trial=2,
            steps=num_steps,
        )

    return f.getvalue().split("\n")


def calculate_gradients(
    batch_size: int = 4,
    epoch_size: int = 64,
    num_epochs: int = 3,
    lr: float = 0.001,
) -> typing.List[float]:
    # independently compute expected metrics
    batches = [
        (v[:], v[:])
        for v in (
            [x * 0.1 + 1.0 for x in range(y, y + batch_size)]
            for y in (z % epoch_size for z in range(0, epoch_size * num_epochs, batch_size))
        )
    ]

    def compute_expected_weight(
        data: typing.List[float], label: typing.List[float], w: float
    ) -> float:
        n = len(data)
        expected_step = 2.0 * lr * sum((d * (l - d * w) for d, l in zip(data, label))) / n
        return w + expected_step

    expected_weights = []
    weight = 0.0
    data: typing.List[float] = []
    label: typing.List[float] = []
    for i, batch in enumerate(batches):
        if i % 2 == 0:
            # for even-numbered batches the optimizer step is a no-op:
            # the weights don't change
            data, label = batch
        else:
            additional_data, additional_label = batch
            data += additional_data
            label += additional_label
            weight = compute_expected_weight(data, label, weight)
        expected_weights.append(weight)

    return expected_weights

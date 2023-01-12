# type: ignore
import os
import pathlib
import random
import sys
import typing

import numpy as np
import pytest
import torch

import determined as det
from determined import gpu, pytorch
from tests.experiment import utils  # noqa: I100
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

    def test_require_global_batch_size(self) -> None:
        bad_hparams = dict(self.hparams)
        del bad_hparams["global_batch_size"]
        with pytest.raises(
            det.errors.InvalidExperimentException, match="is a required hyperparameter"
        ):
            _ = create_trial_and_trial_controller(
                trial_class=pytorch_onevar_model.OneVarTrial,
                hparams=bad_hparams,
            )

    def test_onevar_single(self) -> None:
        """Assert that the training loss and validation error decrease monotonically."""
        trial, trial_controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
        )

        train_steps, metrics = trial_controller._train_with_steps(
            training_enumerator=enumerate(trial_controller.training_iterator),
            train_steps=[
                pytorch._TrainStep(step_type=pytorch._TrainStepType.TRAIN, unit=pytorch.Batch(100))
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

    def test_training_metrics(self) -> None:
        trial, trial_controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialWithTrainingMetrics,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
        )

        train_steps, metrics = trial_controller._train_with_steps(
            training_enumerator=enumerate(trial_controller.training_iterator),
            train_steps=[
                pytorch._TrainStep(step_type=pytorch._TrainStepType.TRAIN, unit=pytorch.Batch(100))
            ],
        )

        assert len(train_steps) == 1, "unexpected train step count"
        assert train_steps[0].limit_reached, "train step did not reach expected limit"
        assert len(metrics) == 100, "metrics length did not match input"
        for metric in metrics:
            assert "mse" in metric

    def test_nonscalar_validation(self) -> None:
        trial, trial_controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialWithNonScalarValidation,
            hparams=self.hparams,
            expose_gpus=True,
            trial_seed=self.trial_seed,
        )

        val_metrics = trial_controller._validate()

        assert "mse" in val_metrics

    def test_checkpointing_and_restoring(self, tmp_path: pathlib.Path) -> None:
        updated_hparams = {
            "lr_scheduler_step_mode": pytorch.LRScheduler.StepMode.STEP_EVERY_BATCH.value,
            **self.hparams,
        }
        self.checkpoint_and_restore(updated_hparams, tmp_path, (100, 100))

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

        tm_a, tm_b = self.checkpoint_and_restore(
            hparams=updated_hparams, tmp_path=tmp_path, steps=(200, 200)
        )

        amp_metrics_test(trial_class, tm_a)
        amp_metrics_test(trial_class, tm_b)

    def test_restore_invalid_checkpoint(self, tmp_path: pathlib.Path) -> None:
        # Build, train, and save a checkpoint with the normal hyperparameters.
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))

        # Trial A: run with 100 batches and checkpoint
        trial_A, trial_controller_A = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=100,
            min_checkpoint_batches=100,
            checkpoint_dir=checkpoint_dir,
        )

        trial_controller_A.run()

        assert len(trial_A.checkpoint_callback.uuids) == 1, "trial did not return a checkpoint UUID"

        # Trial A: restore from checkpoint with invalid hparams
        invalid_hparams = {**self.hparams, "features": 2}
        assert invalid_hparams != self.hparams

        with pytest.raises(RuntimeError):
            trial_A, trial_controller_A = create_trial_and_trial_controller(
                trial_class=pytorch_onevar_model.OneVarTrial,
                hparams=invalid_hparams,
                trial_seed=self.trial_seed,
                max_batches=100,
                min_validation_batches=100,
                min_checkpoint_batches=sys.maxsize,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=trial_A.checkpoint_callback.uuids[0],
                steps_completed=trial_controller_A.state.batches_trained,
            )
            trial_controller_A.run()

    def test_reproducibility(self) -> None:
        training_metrics = {"A": [], "B": []}
        validation_metrics = {"A": [], "B": []}

        # Trial A
        trial_A, trial_controller_A = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=1000,
            min_validation_batches=100,
            min_checkpoint_batches=sys.maxsize,
        )
        trial_controller_A.run()

        metrics_callback = trial_A.metrics_callback
        training_metrics["A"] = metrics_callback.training_metrics
        validation_metrics["A"] = metrics_callback.validation_metrics

        # Trial B
        trial_B, trial_controller_B = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=1000,
            min_validation_batches=100,
            min_checkpoint_batches=sys.maxsize,
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

    def test_custom_eval(self) -> None:
        training_metrics = {"A": [], "B": []}  # type: typing.Dict
        validation_metrics = {"A": [], "B": []}  # type: typing.Dict

        trial_A, trial_controller_A = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=900,
            min_validation_batches=100,
            min_checkpoint_batches=sys.maxsize,
        )
        trial_controller_A.run()

        metrics_callback = trial_A.metrics_callback

        training_metrics["A"] = metrics_callback.training_metrics
        validation_metrics["A"] = metrics_callback.validation_metrics

        trial_B, trial_controller_B = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialCustomEval,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            expose_gpus=True,
            max_batches=900,
            min_validation_batches=100,
            min_checkpoint_batches=sys.maxsize,
        )
        trial_controller_B.run()

        metrics_callback = trial_B.metrics_callback
        training_metrics["B"] = metrics_callback.training_metrics
        validation_metrics["B"] = metrics_callback.validation_metrics

        for original, custom_eval in zip(training_metrics["A"], training_metrics["B"]):
            assert np.allclose(original["loss"], custom_eval["loss"], atol=1e-6)

        for original, custom_eval in zip(validation_metrics["A"], validation_metrics["B"]):
            assert np.allclose(original["val_loss"], custom_eval["val_loss"], atol=1e-6)

    def test_grad_clipping(self) -> None:
        training_metrics = {"original": [], "clipped_by_norm": [], "clipped_by_val": []}
        validation_metrics = {"original": [], "clipped_by_norm": [], "clipped_by_val": []}

        trial, controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialGradClipping,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
        )
        controller.run()

        metrics_callback = trial.metrics_callback

        training_metrics["original"] = metrics_callback.training_metrics
        validation_metrics["original"] = metrics_callback.validation_metrics

        updated_hparams = {"gradient_clipping_l2_norm": 0.0001, **self.hparams}
        trial, controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialGradClipping,
            hparams=updated_hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
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
        trial, controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialGradClipping,
            hparams=updated_hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
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

    def test_per_metric_reducers(self) -> None:
        _, trial_controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialPerMetricReducers,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=2,
            min_validation_batches=1,
            min_checkpoint_batches=sys.maxsize,
        )
        trial_controller.run()

    def test_callbacks(self, tmp_path: pathlib.Path) -> None:
        checkpoint_dir = tmp_path.joinpath("checkpoint")

        hparams1 = dict(self.hparams)
        hparams1["global_batch_size"] = 2
        training_epochs = 2
        num_batches = (
            training_epochs
            * len(pytorch_onevar_model.OnesDataset())
            // hparams1["global_batch_size"]
        )

        trial, trial_controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialCallbacks,
            hparams=hparams1,
            checkpoint_dir=str(checkpoint_dir),
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

        trial, trial_controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialCallbacks,
            hparams=hparams1,
            checkpoint_dir=str(checkpoint_dir),
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

        trial, trial_controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialCallbacks,
            hparams=hparams1,
            checkpoint_dir=str(checkpoint_dir),
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
        lr_scheduler_step_mode,
    ) -> None:
        hparams = self.hparams.copy()
        hparams["lr_scheduler_step_mode"] = lr_scheduler_step_mode
        hparams["global_batch_size"] = 64

        _, controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialAccessContext,
            hparams=hparams,
            trial_seed=self.trial_seed,
            max_batches=1,
            min_validation_batches=1,
            min_checkpoint_batches=sys.maxsize,
        )
        controller.run()

    def test_variable_workload_size(self) -> None:
        trial, controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
        )

        training_metrics = []
        total_steps, total_batches_processed = 10, 0
        for step_id in range(1, total_steps):
            num_batches = step_id
            train_steps, metrics = controller._train_with_steps(
                training_enumerator=enumerate(controller.training_iterator),
                train_steps=[
                    pytorch._TrainStep(
                        step_type=pytorch._TrainStepType.TRAIN, unit=pytorch.Batch(num_batches)
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

    def test_custom_reducers(self) -> None:
        trial, controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=30,
            min_validation_batches=30,
            min_checkpoint_batches=sys.maxsize,
            scheduling_unit=10,
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

    def test_reject_unnamed_nondict_metric(self) -> None:
        trial, controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
        )

        def reducer_fn(_):
            return 1.0

        # Inject an unnamed metric which returns a non-dict (which is not allowed).
        controller.context.wrap_reducer(reducer_fn)

        with pytest.raises(AssertionError, match="name=None but it did not return a dict"):
            controller.run()

    def test_reject_named_dict_metric(self) -> None:
        # If at some point in the future the webui is able to render scalar metrics inside
        # nested dictionary metrics, this test could go away.

        _, controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
        )

        def reducer_fn(_):
            return {"my_metric": 1.0}

        # Inject a named metric which returns a dict (which is not allowed).
        controller.context.wrap_reducer(reducer_fn, name="my_metric")

        with pytest.raises(AssertionError, match="with name set but it returned a dict anyway"):
            controller.run()

    def test_require_disable_dataset_reproducibility(self) -> None:
        hparams = dict(self.hparams)
        hparams["dataloader_type"] = "torch"
        hparams["disable_dataset_reproducibility_checks"] = False

        with pytest.raises(RuntimeError, match="you can disable this check by calling"):
            trial, controller = create_trial_and_trial_controller(
                trial_class=pytorch_onevar_model.OneVarTrial,
                hparams=hparams,
                trial_seed=self.trial_seed,
                max_batches=100,
                min_validation_batches=10,
                min_checkpoint_batches=sys.maxsize,
            )
            controller.run()

    def test_custom_dataloader(self) -> None:
        hparams = dict(self.hparams)
        hparams["dataloader_type"] = "torch"
        hparams["disable_dataset_reproducibility_checks"] = True

        trial, controller = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
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

    def test_gradient_aggregation(self) -> None:
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

        trial, controller = create_trial_and_trial_controller(
            exp_config=exp_config,
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            trial_seed=self.trial_seed,
            max_batches=100,
            min_validation_batches=10,
            min_checkpoint_batches=sys.maxsize,
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
    def test_amp(self, trial_class) -> None:
        """Train a linear model using Determined with Automated Mixed Precision in three ways:
        Using Apex and using PyTorch AMP both "automatically" and "manually". In the "manual" case,
        we use the context manager ``autoscale`` in the model's training and
        evaluating methods; a scaler object is wrapped in a Determined context. The same
        is done under the hood in the first two cases.
        """
        if trial_class is pytorch_onevar_model.OneVarApexAMPTrial and not HAVE_APEX:
            pytest.skip("Apex not available")

        # The assertions logic in make_amp_workloads require a batch size of one
        hparams = dict(self.hparams)
        hparams["global_batch_size"] = 1

        trial, controller = create_trial_and_trial_controller(
            trial_class=trial_class,
            hparams=hparams,
            trial_seed=self.trial_seed,
            expose_gpus=True,
            max_batches=20,
            min_validation_batches=1,
            min_checkpoint_batches=sys.maxsize,
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
    def test_amp_with_gradient_aggregation(self, trial_class) -> None:
        """Similar to test_amp but with gradient aggregation."""
        if trial_class is pytorch_onevar_model.OneVarApexAMPTrial and not HAVE_APEX:
            pytest.skip("Apex not available")

        # The assertions logic in make_amp_workloads require a batch size of one
        hparams = dict(self.hparams)
        hparams["global_batch_size"] = 1

        AGG_FREQ = 2
        exp_config = utils.make_default_exp_config(
            hparams,
            scheduling_unit=1,
            searcher_metric=trial_class._searcher_metric,
        )
        exp_config["optimizations"].update(
            {
                "aggregation_frequency": AGG_FREQ,
                "average_aggregated_gradients": True,
            }
        )

        trial, trial_controller = create_trial_and_trial_controller(
            exp_config=exp_config,
            trial_class=trial_class,
            hparams=hparams,
            trial_seed=self.trial_seed,
            expose_gpus=True,
            max_batches=20 * AGG_FREQ,
            min_validation_batches=1,
            min_checkpoint_batches=sys.maxsize,
        )
        trial_controller.run()

        metrics_callback = trial.metrics_callback
        training_metrics = metrics_callback.training_metrics

        amp_metrics_test(trial_class, training_metrics, agg_freq=AGG_FREQ)

    def test_trainer(self) -> None:
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

    def test_trainer_callbacks(self) -> None:
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

    def checkpoint_and_restore(
        self, hparams: typing.Dict, tmp_path: pathlib.Path, steps: typing.Tuple[int, int] = (1, 1)
    ) -> typing.Tuple[
        typing.Sequence[typing.Dict[str, typing.Any]], typing.Sequence[typing.Dict[str, typing.Any]]
    ]:
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        training_metrics = {"A": [], "B": []}
        validation_metrics = {"A": [], "B": []}

        # Trial A: train 100 batches and checkpoint
        trial_A, trial_controller_A = create_trial_and_trial_controller(
            trial_class=pytorch_onevar_model.OneVarTrialWithLRScheduler,
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
            trial_class=pytorch_onevar_model.OneVarTrialWithLRScheduler,
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
            trial_class=pytorch_onevar_model.OneVarTrialWithLRScheduler,
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


@pytest.mark.parametrize(
    "ckpt,istrial",
    [
        ("0.13.13-pytorch-old", False),
        ("0.13.13-pytorch-flex", True),
        ("0.17.6-pytorch", True),
        ("0.17.7-pytorch", True),
    ],
)
def test_checkpoint_loading(ckpt: str, istrial: bool):
    checkpoint_dir = os.path.join(utils.fixtures_path("ancient-checkpoints"), f"{ckpt}")
    trial = pytorch.load_trial_from_checkpoint_path(checkpoint_dir)
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
        loss = metrics["loss"].item()
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


def create_trial_and_trial_controller(
    trial_class: pytorch.PyTorchTrial,
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
            steps_completed=steps_completed,
            managed_training=False,
            debug_enabled=False,
        )
        trial_context._set_gradient_compression(False)
        trial_context._set_average_aggregated_gradients(True)
        trial_inst = trial_class(trial_context)

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

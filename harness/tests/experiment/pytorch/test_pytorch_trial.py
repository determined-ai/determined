# type: ignore
import os
import pathlib
import typing

import numpy as np
import pytest
import torch

import determined as det
from determined import pytorch, workload
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import pytorch_onevar_model, pytorch_xor_model

# Apex is included only for GPU trials.
try:
    import apex

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
        utils.ensure_requires_global_batch_size(pytorch_onevar_model.OneVarTrial, self.hparams)

    def test_onevar_single(self) -> None:
        """Assert that the training loss and validation error decrease monotonically."""

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=100, validation_freq=10)
            training_metrics, validation_metrics = trainer.result()

            # Check the gradient update at every step.
            for idx, batch_metrics in enumerate(training_metrics):
                pytorch_onevar_model.OneVarTrial.check_batch_metrics(batch_metrics, idx)

            # We expect the validation error and training loss to be
            # monotonically decreasing.
            for older, newer in zip(training_metrics, training_metrics[1:]):
                assert newer["loss"] <= older["loss"]

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_xor_multi_validation(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10)
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "binary_error" in metrics
                assert "accuracy" in metrics

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialWithMultiValidation,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_xor_training_metrics(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10)
            training_metrics, validation_metrics = trainer.result()

            for metrics in training_metrics:
                assert "accuracy" in metrics

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialWithTrainingMetrics,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_xor_nonscalar_validation(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10)
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "binary_error" in metrics
                assert "predictions" in metrics

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialWithNonScalarValidation,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
            expose_gpus=True,
        )
        controller.run()

    def test_checkpointing_and_restoring(self, tmp_path: pathlib.Path) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream,
            checkpoint_dir: typing.Optional[str] = None,
            latest_checkpoint: typing.Optional[str] = None,
            steps_completed: int = 0,
        ) -> det.TrialController:
            updated_hparams = {
                "lr_scheduler_step_mode": pytorch.LRScheduler.StepMode.STEP_EVERY_BATCH.value,
                **self.hparams,
            }
            return utils.make_trial_controller_from_trial_implementation(
                trial_class=pytorch_xor_model.XORTrialWithLRScheduler,
                hparams=updated_hparams,
                workloads=workloads,
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
                steps_completed=steps_completed,
            )

        utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

    def test_restore_invalid_checkpoint(self, tmp_path: pathlib.Path) -> None:
        # Build, train, and save a checkpoint with the normal hyperparameters.
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        latest_checkpoint = None
        steps_completed = 0

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1)
            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint, steps_completed
            latest_checkpoint = interceptor.metrics_result()["uuid"]
            steps_completed = trainer.get_steps_completed()

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialMulti,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
            checkpoint_dir=checkpoint_dir,
        )
        controller.run()

        # Verify that an invalid architecture fails to load from the checkpoint.
        def make_invalid_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1)

        invalid_hparams = {"hidden_size": 3, "learning_rate": 0.5, "global_batch_size": 4}
        assert invalid_hparams != self.hparams

        with pytest.raises(RuntimeError):
            invalid_controller = utils.make_trial_controller_from_trial_implementation(
                trial_class=pytorch_xor_model.XORTrialMulti,
                hparams=invalid_hparams,
                workloads=make_invalid_workloads(),
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
                steps_completed=steps_completed,
            )
            invalid_controller.run()

    def test_reproducibility(self) -> None:
        def controller_fn(workloads: workload.Stream) -> det.TrialController:
            return utils.make_trial_controller_from_trial_implementation(
                trial_class=pytorch_xor_model.XORTrial,
                hparams=self.hparams,
                workloads=workloads,
                trial_seed=self.trial_seed,
            )

        _ = utils.reproducibility_test(controller_fn, steps=1000, validation_freq=100)

    def test_custom_eval(self) -> None:
        training_metrics = {}
        validation_metrics = {}

        def make_workloads(tag: str) -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=900, validation_freq=100)
            tm, vm = trainer.result()
            training_metrics[tag] = tm
            validation_metrics[tag] = vm

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrial,
            hparams=self.hparams,
            workloads=make_workloads("A"),
            trial_seed=self.trial_seed,
        )
        controller.run()

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialCustomEval,
            hparams=self.hparams,
            workloads=make_workloads("B"),
            trial_seed=self.trial_seed,
            expose_gpus=True,
        )
        controller.run()

        for original, custom_eval in zip(training_metrics["A"], training_metrics["B"]):
            assert original["loss"] == custom_eval["loss"]

        for original, custom_eval in zip(validation_metrics["A"], validation_metrics["B"]):
            assert original["loss"] == custom_eval["loss"]

    def test_grad_clipping(self) -> None:
        training_metrics = {}
        validation_metrics = {}

        def make_workloads(tag: str) -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=1000, validation_freq=100)
            tm, vm = trainer.result()
            training_metrics[tag] = tm
            validation_metrics[tag] = vm

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialGradClipping,
            hparams=self.hparams,
            workloads=make_workloads("original"),
            trial_seed=self.trial_seed,
        )
        controller.run()

        updated_hparams = {"gradient_clipping_l2_norm": 0.0001, **self.hparams}
        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialGradClipping,
            hparams=updated_hparams,
            workloads=make_workloads("clipped_by_norm"),
            trial_seed=self.trial_seed,
        )
        controller.run()

        for idx, (original, clipped) in enumerate(
            zip(training_metrics["original"], training_metrics["clipped_by_norm"])
        ):
            if idx < 10:
                continue
            assert original["loss"] != clipped["loss"]

        updated_hparams = {"gradient_clipping_value": 0.0001, **self.hparams}
        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialGradClipping,
            hparams=updated_hparams,
            workloads=make_workloads("clipped_by_val"),
            trial_seed=self.trial_seed,
        )
        controller.run()

        for idx, (original, clipped) in enumerate(
            zip(training_metrics["original"], training_metrics["clipped_by_val"])
        ):
            if idx < 10:
                continue
            assert original["loss"] != clipped["loss"]

    def test_per_metric_reducers(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=2, validation_freq=1, scheduling_unit=1)

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialPerMetricReducers,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_callbacks(self, tmp_path: pathlib.Path) -> None:
        checkpoint_dir = tmp_path.joinpath("checkpoint")
        latest_checkpoint = None
        steps_completed = 0

        controller = None

        def make_workloads1() -> workload.Stream:
            nonlocal controller
            assert controller.trial.counter.trial_startups == 1

            yield workload.train_workload(1, 1, 0, 4), workload.ignore_workload_response
            assert controller is not None, "controller was never set!"
            assert controller.trial.counter.__dict__ == {
                "trial_startups": 1,
                "validation_steps_started": 0,
                "validation_steps_ended": 0,
                "checkpoints_written": 0,
                "checkpoints_uploaded": 0,
                "training_started_times": 1,
                "training_epochs_started": 2,
                "training_epochs_ended": 2,
                "training_workloads_ended": 1,
                "trial_shutdowns": 0,
            }
            assert controller.trial.legacy_counter.__dict__ == {
                "legacy_on_training_epochs_start_calls": 2
            }

            yield workload.validation_workload(), workload.ignore_workload_response
            assert controller.trial.counter.__dict__ == {
                "trial_startups": 1,
                "validation_steps_started": 1,
                "validation_steps_ended": 1,
                "checkpoints_written": 0,
                "checkpoints_uploaded": 0,
                "training_started_times": 1,
                "training_epochs_started": 2,
                "training_epochs_ended": 2,
                "training_workloads_ended": 1,
                "trial_shutdowns": 0,
            }
            assert controller.trial.legacy_counter.__dict__ == {
                "legacy_on_training_epochs_start_calls": 2
            }

            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint, steps_completed
            latest_checkpoint = interceptor.metrics_result()["uuid"]
            steps_completed = 1
            assert controller.trial.counter.__dict__ == {
                "trial_startups": 1,
                "validation_steps_started": 1,
                "validation_steps_ended": 1,
                "checkpoints_written": 1,
                "checkpoints_uploaded": 1,
                "training_started_times": 1,
                "training_epochs_started": 2,
                "training_epochs_ended": 2,
                "training_workloads_ended": 1,
                "trial_shutdowns": 0,
            }
            assert controller.trial.legacy_counter.__dict__ == {
                "legacy_on_training_epochs_start_calls": 2
            }

        hparams1 = dict(self.hparams)
        hparams1["global_batch_size"] = 2
        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialCallbacks,
            hparams=hparams1,
            workloads=make_workloads1(),
            checkpoint_dir=str(checkpoint_dir),
        )
        controller.run()
        assert controller.trial.counter.trial_shutdowns == 1

        # Verify the checkpoint loading callback works.

        def make_workloads2() -> workload.Stream:
            yield workload.train_workload(1, 1, 0), workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialCallbacks,
            hparams=self.hparams,
            workloads=make_workloads2(),
            checkpoint_dir=str(checkpoint_dir),
            latest_checkpoint=latest_checkpoint,
            steps_completed=steps_completed,
        )
        controller.run()
        assert controller.trial.counter.__dict__ == {
            # Note: trial_startups will get reset by the loading logic.
            "trial_startups": 1,
            "validation_steps_started": 1,
            "validation_steps_ended": 1,
            # Note: checkpoints_written, checkpoints_uploaded, and trial_shutdowns, cannot be
            # persisted, as they are all updated after checkpointing.
            "checkpoints_written": 0,
            "checkpoints_uploaded": 0,
            "training_started_times": 2,
            "training_epochs_started": 3,
            "training_epochs_ended": 3,
            "training_workloads_ended": 2,
            "trial_shutdowns": 1,
        }
        assert controller.trial.legacy_counter.__dict__ == {
            "legacy_on_training_epochs_start_calls": 1
        }

    @pytest.mark.parametrize(
        "lr_scheduler_step_mode", [mode.value for mode in pytorch.LRScheduler.StepMode]
    )
    def test_context(
        self,
        lr_scheduler_step_mode,
    ) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1, scheduling_unit=1)

        hparams = self.hparams.copy()
        hparams["lr_scheduler_step_mode"] = lr_scheduler_step_mode

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialAccessContext,
            hparams=hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_variable_workload_size(self) -> None:
        def make_workloads() -> workload.Stream:
            training_metrics = []
            interceptor = workload.WorkloadResponseInterceptor()

            total_steps, total_batches_processed = 10, 0
            for step_id in range(1, total_steps):
                num_batches = step_id
                yield from interceptor.send(
                    workload.train_workload(
                        step_id,
                        num_batches=num_batches,
                        total_batches_processed=total_batches_processed,
                    ),
                )
                metrics = interceptor.metrics_result()
                batch_metrics = metrics["metrics"]["batch_metrics"]
                assert len(batch_metrics) == num_batches, "did not run for expected num_batches"
                training_metrics.extend(batch_metrics)
                total_batches_processed += num_batches

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrial,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_custom_reducers(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=3, validation_freq=3, scheduling_unit=10)
            training_metrics = trainer.get_avg_training_metrics()
            _, validation_metrics = trainer.result()

            batch_size = self.hparams["global_batch_size"]

            for i, metrics in enumerate(training_metrics):
                expect = pytorch_onevar_model.TriangleLabelSum.expect(
                    batch_size, 10 * i, 10 * (i + 1)
                )
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

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_reject_unnamed_nondict_metric(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1, scheduling_unit=1)

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )

        def reducer_fn(_):
            return 1.0

        # Inject an unnamed metric which returns a non-dict (which is not allowed).
        controller.context.wrap_reducer(reducer_fn, name=None)

        with pytest.raises(AssertionError, match="name=None but it did not return a dict"):
            controller.run()

    def test_reject_named_dict_metric(self) -> None:
        # If at some point in the future the webui is able to render scalar metrics inside
        # nested dictionary metrics, this test could go away.

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1, scheduling_unit=1)

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )

        def reducer_fn(_):
            return {"my_metric": 1.0}

        # Inject a named metric which returns a dict (which is not allowed).
        controller.context.wrap_reducer(reducer_fn, name="my_metric")

        with pytest.raises(AssertionError, match="with name set but it returned a dict anyway"):
            controller.run()

    def test_require_disable_dataset_reproducibility(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1, scheduling_unit=1)
            yield workload.terminate_workload(), [], workload.ignore_workload_response

        hparams = dict(self.hparams)
        hparams["dataloader_type"] = "torch"
        hparams["disable_dataset_reproducibility_checks"] = False

        with pytest.raises(RuntimeError, match="you can disable this check by calling"):
            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class=pytorch_onevar_model.OneVarTrial,
                hparams=hparams,
                workloads=make_workloads(),
                trial_seed=self.trial_seed,
            )
            controller.run()

    def test_custom_dataloader(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=100, validation_freq=10)
            training_metrics, validation_metrics = trainer.result()

            # Check the gradient update at every step.
            for idx, batch_metrics in enumerate(training_metrics):
                pytorch_onevar_model.OneVarTrial.check_batch_metrics(batch_metrics, idx)

            # We expect the validation error and training loss to be
            # monotonically decreasing.
            for older, newer in zip(training_metrics, training_metrics[1:]):
                assert newer["loss"] <= older["loss"]

        hparams = dict(self.hparams)
        hparams["dataloader_type"] = "torch"
        hparams["disable_dataset_reproducibility_checks"] = True

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    @pytest.mark.skipif(not torch.cuda.is_available(), reason="no gpu available")
    @pytest.mark.gpu
    @pytest.mark.parametrize(
        "trial_class,assert_output_float16,assert_scale_changed",
        [
            (pytorch_onevar_model.OneVarApexAMPTrial, False, True),
            (pytorch_onevar_model.OneVarAutoAMPTrial, False, False),
            (pytorch_onevar_model.OneVarManualAMPTrial, True, True),
        ],
        ids=[
            "apex",
            "autocast",
            "manual",
        ],
    )
    def test_amp(
        self,
        trial_class,
        assert_output_float16,
        assert_scale_changed,
    ) -> None:
        if trial_class is pytorch_onevar_model.OneVarApexAMPTrial and not HAVE_APEX:
            pytest.skip("Apex not available")

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=trial_class,
            hparams=self.hparams,
            workloads=make_amp_workloads(assert_output_float16, assert_scale_changed),
            trial_seed=self.trial_seed,
            expose_gpus=True,
        )
        controller.run()


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


def make_amp_workloads(
    assert_output_float16=False,
    assert_scale_changed=False,
) -> workload.Stream:
    trainer = utils.TrainAndValidate()
    yield from trainer.send(steps=10, validation_freq=1, scheduling_unit=1)
    training_metrics, _ = trainer.result()
    scale_ever_decreased = False
    scale_ever_increased = False
    for older, newer in zip(training_metrics, training_metrics[1:]):
        assert newer["loss"] <= older["loss"]
        assert newer["loss"].dtype is np.dtype("float32")
        if assert_output_float16:
            assert newer["output"].dtype is np.dtype("float16")
        else:
            # Automatic usages of Apex or native AMP cast the output back to float32
            # For the latter case, see the hook end_f16
            #   defined in PyTorchTrialContext.autocast_forward_pass
            assert newer["output"].dtype is np.dtype("float32")
        if assert_scale_changed:
            scale_ever_decreased = scale_ever_decreased or newer["scale"] < older["scale"]
            scale_ever_increased = scale_ever_increased or newer["scale"] > older["scale"]
    if assert_scale_changed:
        # TODO: change test so that we can assert both these flags are True
        assert scale_ever_decreased or scale_ever_increased

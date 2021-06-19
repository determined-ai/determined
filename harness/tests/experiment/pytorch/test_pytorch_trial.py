import os
import pathlib
import subprocess
import sys
import typing

import pytest
import torch

import determined as det
from determined import pytorch, workload
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import pytorch_onevar_model, pytorch_xor_model


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
        }

    def test_onevar_single(self) -> None:
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

            yield workload.terminate_workload(), workload.ignore_workload_response

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

            yield workload.terminate_workload(), workload.ignore_workload_response

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

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialWithTrainingMetrics,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    @pytest.mark.usefixtures("expose_gpus")
    def test_xor_nonscalar_validation(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10)
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "binary_error" in metrics
                assert "predictions" in metrics

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialWithNonScalarValidation,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_checkpointing_and_restoring(self, tmp_path: pathlib.Path) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream,
            checkpoint_dir: typing.Optional[str] = None,
            latest_checkpoint: typing.Optional[typing.Dict[str, typing.Any]] = None,
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
            )

        utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

    def test_restore_invalid_checkpoint(self, tmp_path: pathlib.Path) -> None:
        # Build, train, and save a checkpoint with the normal hyperparameters.
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        latest_checkpoint = None

        def make_workloads_1() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1)
            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint
            latest_checkpoint = interceptor.metrics_result()["metrics"].__json__()
            yield workload.terminate_workload(), workload.ignore_workload_response

        controller1 = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialMulti,
            hparams=self.hparams,
            workloads=make_workloads_1(),
            trial_seed=self.trial_seed,
            checkpoint_dir=checkpoint_dir,
        )
        controller1.run()

        # Verify that an invalid architecture fails to load from the checkpoint.
        def make_workloads_2() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1)
            yield workload.terminate_workload(), workload.ignore_workload_response

        hparams2 = {"hidden_size": 3, "learning_rate": 0.5, "global_batch_size": 4}

        with pytest.raises(RuntimeError):
            controller2 = utils.make_trial_controller_from_trial_implementation(
                trial_class=pytorch_xor_model.XORTrialMulti,
                hparams=hparams2,
                workloads=make_workloads_2(),
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
            )
            controller2.run()

    def test_reproducibility(self) -> None:
        def controller_fn(workloads: workload.Stream) -> det.TrialController:
            return utils.make_trial_controller_from_trial_implementation(
                trial_class=pytorch_xor_model.XORTrial,
                hparams=self.hparams,
                workloads=workloads,
                trial_seed=self.trial_seed,
            )

        utils.reproducibility_test(controller_fn, steps=1000, validation_freq=100)

    @pytest.mark.usefixtures("expose_gpus")
    def test_custom_eval(self) -> None:
        training_metrics = {}
        validation_metrics = {}

        def make_workloads(tag: str) -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=900, validation_freq=100)
            tm, vm = trainer.result()
            training_metrics[tag] = tm
            validation_metrics[tag] = vm

            yield workload.terminate_workload(), workload.ignore_workload_response

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

            yield workload.terminate_workload(), workload.ignore_workload_response

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
            yield workload.terminate_workload(), workload.ignore_workload_response

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

        controller = None  # type: ignore

        def make_workloads1() -> workload.Stream:
            nonlocal controller

            yield workload.train_workload(1, 1, 0), workload.ignore_workload_response
            assert controller is not None, "controller was never set!"
            assert controller.trial.counter.__dict__ == {
                "validation_steps_started": 0,
                "validation_steps_ended": 0,
                "checkpoints_ended": 0,
            }

            yield workload.validation_workload(), workload.ignore_workload_response
            assert controller.trial.counter.__dict__ == {
                "validation_steps_started": 1,
                "validation_steps_ended": 1,
                "checkpoints_ended": 0,
            }

            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint
            latest_checkpoint = interceptor.metrics_result()["metrics"].__json__()
            assert controller.trial.counter.__dict__ == {
                "validation_steps_started": 1,
                "validation_steps_ended": 1,
                "checkpoints_ended": 1,
            }

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialCallbacks,
            hparams=self.hparams,
            workloads=make_workloads1(),
            checkpoint_dir=str(checkpoint_dir),
        )
        controller.run()

        # Verify the checkpoint loading callback works.

        def make_workloads2() -> workload.Stream:
            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialCallbacks,
            hparams=self.hparams,
            workloads=make_workloads2(),
            checkpoint_dir=str(checkpoint_dir),
            latest_checkpoint=latest_checkpoint,
        )
        controller.run()
        assert controller.trial.counter.__dict__ == {
            "validation_steps_started": 1,
            "validation_steps_ended": 1,
            "checkpoints_ended": 0,
        }

    def test_context(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1, scheduling_unit=1)
            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialAccessContext,
            hparams=self.hparams,
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

            yield workload.terminate_workload(), workload.ignore_workload_response

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

            yield workload.terminate_workload(), workload.ignore_workload_response

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
            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_onevar_model.OneVarTrial,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )

        def reducer_fn(_):
            return 1.0

        # Inject an unnamed metric which returns a non-dict (which is not allowed).
        controller.context.wrap_reducer(reducer_fn)

        with pytest.raises(AssertionError, match="name=None but it did not return a dict"):
            controller.run()

    def test_reject_named_dict_metric(self) -> None:
        # If at some point in the future the webui is able to render scalar metrics inside of
        # nested dictionary metrics, this test could go away.

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1, scheduling_unit=1)
            yield workload.terminate_workload(), workload.ignore_workload_response

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


def test_create_trial_instance() -> None:
    utils.create_trial_instance(pytorch_xor_model.XORTrial)


def test_native_api_local_test() -> None:
    subprocess.check_call(
        args=[sys.executable, "pytorch_onevar_model.py"],
        cwd=utils.fixtures_path(""),
        env={
            "PYTHONUNBUFFERED": "1",
            "PYTHONPATH": f"$PYTHONPATH:{utils.repo_path('harness')}",
            **os.environ,
        },
    )

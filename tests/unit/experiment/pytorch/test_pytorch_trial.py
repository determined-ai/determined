import pathlib
import typing

import pytest
import torch

import determined as det
from determined import workload
from determined_common import check
from tests.unit.experiment import utils  # noqa: I100
from tests.unit.experiment.fixtures import pytorch_xor_model


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
        self.hparams = {"hidden_size": 2, "learning_rate": 0.5, "global_batch_size": 4}

    def test_xor_single(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=1000, validation_freq=100)
            training_metrics, validation_metrics = trainer.result()

            # We expect the validation error and training loss to be
            # monotonically decreasing.
            for older, newer in zip(training_metrics, training_metrics[1:]):
                assert newer["loss"] <= older["loss"]

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrial,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_xor_multi(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=1000, validation_freq=100)
            training_metrics, validation_metrics = trainer.result()

            # We expect the validation error and training loss to be
            # monotonically decreasing.
            for older, newer in zip(training_metrics, training_metrics[1:]):
                assert newer["loss"] <= older["loss"]

            for older, newer in zip(validation_metrics, validation_metrics[1:]):
                assert newer["binary_error"] <= older["binary_error"]

            assert validation_metrics[-1]["binary_error"] == pytest.approx(0.0)

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialMulti,
            workloads=make_workloads(),
            hparams=self.hparams,
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

            yield workload.terminate_workload(), [], workload.ignore_workload_response

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

            yield workload.terminate_workload(), [], workload.ignore_workload_response

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

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialWithNonScalarValidation,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_checkpointing(self, tmp_path: pathlib.Path) -> None:
        checkpoint_dir = tmp_path.joinpath("checkpoint")

        old_error = -1

        def make_workloads_1() -> workload.Stream:
            nonlocal old_error

            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10)
            training_metrics, validation_metrics = trainer.result()
            old_error = validation_metrics[-1]["binary_error"]

            yield workload.checkpoint_workload(), [
                checkpoint_dir
            ], workload.ignore_workload_response

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialMulti,
            hparams=self.hparams,
            workloads=make_workloads_1(),
            trial_seed=self.trial_seed,
        )
        controller.run()

        # Restore the checkpoint on a new trial instance and recompute
        # validation. The validation error should be the same as it was
        # previously.
        def make_workloads_2() -> workload.Stream:
            interceptor = workload.WorkloadResponseInterceptor()

            yield from interceptor.send(workload.validation_workload(), [])
            metrics = interceptor.metrics_result()

            new_error = metrics["validation_metrics"]["binary_error"]
            assert new_error == pytest.approx(old_error)

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialMulti,
            hparams=self.hparams,
            workloads=make_workloads_2(),
            load_path=checkpoint_dir,
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_fail_restore_invalid_checkpoint(self, tmp_path: pathlib.Path) -> None:
        # Build, train, and save a checkpoint with the normal hyperparameters.
        checkpoint_dir = tmp_path.joinpath("checkpoint")

        def make_workloads_1() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1)
            yield workload.checkpoint_workload(), [
                checkpoint_dir
            ], workload.ignore_workload_response
            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller1 = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialMulti,
            hparams=self.hparams,
            workloads=make_workloads_1(),
            trial_seed=self.trial_seed,
        )
        controller1.run()

        # Verify that an invalid architecture fails to load from the checkpoint.
        def make_workloads_2() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1)
            yield workload.checkpoint_workload(), [
                checkpoint_dir
            ], workload.ignore_workload_response
            yield workload.terminate_workload(), [], workload.ignore_workload_response

        hparams2 = {"hidden_size": 3, "learning_rate": 0.5, "global_batch_size": 4}

        with pytest.raises(RuntimeError):
            controller2 = utils.make_trial_controller_from_trial_implementation(
                trial_class=pytorch_xor_model.XORTrialMulti,
                hparams=hparams2,
                workloads=make_workloads_2(),
                load_path=checkpoint_dir,
                trial_seed=self.trial_seed,
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

        utils.reproducibility_test(
            controller_fn, steps=1000, validation_freq=100,
        )

    def test_optimizer_state(self, tmp_path: pathlib.Path) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream, load_path: typing.Optional[str] = None
        ) -> det.TrialController:
            return utils.make_trial_controller_from_trial_implementation(
                trial_class=pytorch_xor_model.XORTrialOptimizerState,
                hparams=self.hparams,
                workloads=workloads,
                load_path=load_path,
                trial_seed=self.trial_seed,
            )

        utils.optimizer_state_test(make_trial_controller_fn, tmp_path)

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

            yield workload.terminate_workload(), [], workload.ignore_workload_response

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

    def test_lr_schedule_and_lr_checkpoint(self, tmp_path: pathlib.Path) -> None:
        checkpoint_dir = tmp_path.joinpath("checkpoint")
        training_metrics = []

        def make_workloads(checkpoint_dir: str = "") -> workload.Stream:
            nonlocal training_metrics

            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10, batches_per_step=1)
            tm, _ = trainer.result()
            training_metrics += tm

            if checkpoint_dir:
                yield workload.checkpoint_workload(), [
                    checkpoint_dir
                ], workload.ignore_workload_response

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialRestoreLR,
            hparams=self.hparams,
            workloads=make_workloads(checkpoint_dir),
            trial_seed=self.trial_seed,
        )
        controller.run()

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialRestoreLR,
            hparams=self.hparams,
            workloads=make_workloads(),
            load_path=checkpoint_dir,
            trial_seed=self.trial_seed,
        )
        controller.run()

        lrs = [metric["lr"] for metric in training_metrics]
        for i in range(1, len(lrs)):
            assert lrs[i] == lrs[i - 1] + 1

    def test_lr_schedule_user_modify_fail(self, tmp_path: pathlib.Path) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=10, validation_freq=10, batches_per_step=1)
            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialUserStepLRFail,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        with pytest.raises(check.CheckFailedError):
            controller.run()

    def test_lr_schedule_user_modify(self, tmp_path: pathlib.Path) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=10, validation_freq=10, batches_per_step=1)
            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialUserStepLR,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_lr_schedule_step_epoch(self, tmp_path: pathlib.Path) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=10, validation_freq=10, batches_per_step=1)
            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialStepEveryEpoch,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    def test_grad_clipping(self) -> None:
        training_metrics = {}
        validation_metrics = {}

        def make_workloads(tag: str) -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=1000, validation_freq=100)
            tm, vm = trainer.result()
            training_metrics[tag] = tm
            validation_metrics[tag] = vm

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialMulti,
            hparams=self.hparams,
            workloads=make_workloads("original"),
            trial_seed=self.trial_seed,
        )
        controller.run()

        updated_hparams = {"clip_grad_l2_norm": 0.0001, **self.hparams}
        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialMulti,
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

        updated_hparams = {"clip_grad_val": 0.0001, **self.hparams}
        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialMulti,
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
            yield from trainer.send(steps=2, validation_freq=1, batches_per_step=1)
            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=pytorch_xor_model.XORTrialPerMetricReducers,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()


def test_create_trial_instance() -> None:
    utils.create_trial_instance(pytorch_xor_model.XORTrial)

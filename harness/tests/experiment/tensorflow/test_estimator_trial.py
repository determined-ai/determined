import os
import tempfile
from pathlib import Path
from typing import Any, Callable, Dict, Optional

import pytest
import tensorflow as tf

import determined as det
from determined import workload
from determined.exec import harness
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import estimator_linear_model, estimator_xor_model


@pytest.fixture(
    scope="function",
    params=[
        estimator_xor_model.XORTrial,
        estimator_xor_model.XORTrialDataLayer,
    ],
)
def xor_trial_controller(request):
    """
    This fixture will provide a function that takes a hyperparameters dictionary as input and
    returns a trial controller. It is parameterized over different implementations, so that any test
    that uses it may test a full set of implementations.
    """

    def _xor_trial_controller(
        hparams: Dict[str, Any],
        workloads: workload.Stream,
        scheduling_unit: int = 1,
        exp_config: Optional[Dict] = None,
        checkpoint_dir: Optional[str] = None,
        latest_checkpoint: Optional[Dict[str, Any]] = None,
    ) -> det.TrialController:
        return utils.make_trial_controller_from_trial_implementation(
            trial_class=request.param,
            hparams=hparams,
            workloads=workloads,
            scheduling_unit=scheduling_unit,
            exp_config=exp_config,
            trial_seed=325,
            checkpoint_dir=checkpoint_dir,
            latest_checkpoint=latest_checkpoint,
        )

    return _xor_trial_controller


class TestXORTrial:
    def setup_method(self) -> None:
        self.hparams = {
            "hidden_size": 2,
            "learning_rate": 0.1,
            "global_batch_size": 4,
            "optimizer": "sgd",
            "shuffle": False,
        }

    def teardown_method(self) -> None:
        # Cleanup leftover environment variable state.
        for key in harness.ENVIRONMENT_VARIABLE_KEYS:
            if key in os.environ:
                del os.environ[key]

    def test_xor_training(self, xor_trial_controller: Callable) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=5, scheduling_unit=1000)
            training_metrics, validation_metrics = trainer.result()

            # We expect the training loss to be monotonically decreasing and the
            # accuracy to be monotonically increasing.
            for older, newer in zip(training_metrics, training_metrics[1:]):
                assert newer["loss"] < older["loss"]

            for older, newer in zip(validation_metrics, validation_metrics[1:]):
                assert newer["accuracy"] >= older["accuracy"]

            # The final accuracy should be 100%.
            assert validation_metrics[-1]["accuracy"] == pytest.approx(1.0)

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = xor_trial_controller(self.hparams, make_workloads(), scheduling_unit=1000)
        controller.run()

    def test_reproducibility(self, xor_trial_controller: Callable) -> None:
        def controller_fn(workloads: workload.Stream) -> det.TrialController:
            return xor_trial_controller(self.hparams, workloads, scheduling_unit=100)

        utils.reproducibility_test(
            controller_fn=controller_fn, steps=3, validation_freq=1, scheduling_unit=100
        )

    def test_checkpointing(self, tmp_path: Path, xor_trial_controller: Callable) -> None:
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        latest_checkpoint = None
        old_loss = -1

        def make_workloads_1() -> workload.Stream:
            nonlocal old_loss

            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=1, validation_freq=1, scheduling_unit=10)
            training_metrics, validation_metrics = trainer.result()
            old_loss = validation_metrics[-1]["loss"]

            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint
            latest_checkpoint = interceptor.metrics_result()["metrics"].__json__()

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = xor_trial_controller(
            self.hparams,
            make_workloads_1(),
            scheduling_unit=10,
            checkpoint_dir=checkpoint_dir,
        )
        controller.run()

        # Restore the checkpoint on a new trial instance and recompute
        # validation. The validation error should be the same as it was
        # previously.
        def make_workloads_2() -> workload.Stream:
            interceptor = workload.WorkloadResponseInterceptor()

            yield from interceptor.send(workload.validation_workload())
            metrics = interceptor.metrics_result()

            new_loss = metrics["metrics"]["validation_metrics"]["loss"]
            assert new_loss == pytest.approx(old_loss)

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = xor_trial_controller(
            self.hparams,
            make_workloads_2(),
            scheduling_unit=10,
            checkpoint_dir=checkpoint_dir,
            latest_checkpoint=latest_checkpoint,
        )
        controller.run()

    def test_checkpointing_with_serving_fn(
        self, tmp_path: Path, xor_trial_controller: Callable
    ) -> None:
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        latest_checkpoint = None

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1, scheduling_unit=10)

            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint
            latest_checkpoint = interceptor.metrics_result()["metrics"].__json__()

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = xor_trial_controller(
            self.hparams,
            make_workloads(),
            scheduling_unit=10,
            checkpoint_dir=checkpoint_dir,
        )
        controller.run()

        def load_saved_model(path: str) -> None:
            with tf.compat.v1.Session(graph=tf.Graph()) as sess:
                tf.compat.v1.saved_model.loader.load(
                    sess, [tf.compat.v1.saved_model.tag_constants.SERVING], path
                )

        # Determined should export the SavedModel to a subdirectory named "inference"
        # in the checkpoint directory. Within the "inference" subdirectory,
        # there should be a single timestamped subdirectory that contains the
        # exported SavedModel.
        export_path = os.path.join(checkpoint_dir, latest_checkpoint["uuid"], "inference")
        assert os.path.exists(export_path)
        _, dirs, _ = next(os.walk(export_path))
        assert len(dirs) == 1
        load_saved_model(os.path.join(export_path, dirs[0]))

    def test_optimizer_state(self, tmp_path: Path, xor_trial_controller: Callable) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream,
            checkpoint_dir: Optional[str] = None,
            latest_checkpoint: Optional[Dict[str, Any]] = None,
        ) -> det.TrialController:
            hparams = {**self.hparams, "optimizer": "adam"}
            return xor_trial_controller(
                hparams,
                workloads,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
            )

        utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

    def test_hooks(self) -> None:
        with tempfile.TemporaryDirectory() as temp_directory:
            scheduling_unit = 5
            steps = 10
            validation_freq = 5

            def make_workloads() -> workload.Stream:
                trainer = utils.TrainAndValidate()

                yield from trainer.send(
                    steps=steps, validation_freq=validation_freq, scheduling_unit=scheduling_unit
                )
                yield workload.terminate_workload(), workload.ignore_workload_response

            hparams = self.hparams.copy()
            hparams["training_log_path"] = os.path.join(temp_directory, "training.log")
            hparams["val_log_path"] = os.path.join(temp_directory, "val.log")

            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class=estimator_xor_model.XORTrialWithHooks,
                hparams=hparams,
                workloads=make_workloads(),
                scheduling_unit=scheduling_unit,
            )
            controller.run()

            with open(hparams["training_log_path"], "r") as fp:
                assert int(fp.readline()) == scheduling_unit * steps

            with open(hparams["val_log_path"], "r") as fp:
                assert int(fp.readline()) == steps / validation_freq

    def test_custom_hook(self, tmp_path: Path) -> None:
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        latest_checkpoint = None

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=5, scheduling_unit=5)

            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint
            latest_checkpoint = interceptor.metrics_result()["metrics"].__json__()

            yield workload.terminate_workload(), workload.ignore_workload_response

        def verify_callback(checkpoint_dir: str, checkpoint_num: int) -> None:
            with open(os.path.join(checkpoint_dir, "custom.log"), "r") as fp:
                assert int(fp.readline()) == checkpoint_num

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=estimator_xor_model.XORTrialWithCustomHook,
            hparams=self.hparams,
            workloads=make_workloads(),
            scheduling_unit=5,
            checkpoint_dir=checkpoint_dir,
        )
        controller.run()
        verify_callback(os.path.join(checkpoint_dir, latest_checkpoint["uuid"]), checkpoint_num=1)

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=estimator_xor_model.XORTrialWithCustomHook,
            hparams=self.hparams,
            workloads=make_workloads(),
            scheduling_unit=5,
            checkpoint_dir=checkpoint_dir,
            latest_checkpoint=latest_checkpoint,
        )
        controller.run()
        verify_callback(os.path.join(checkpoint_dir, latest_checkpoint["uuid"]), checkpoint_num=2)

    def test_end_of_training_hook(self):
        with tempfile.TemporaryDirectory() as temp_directory:

            def make_workloads() -> workload.Stream:
                trainer = utils.TrainAndValidate()

                yield from trainer.send(steps=2, validation_freq=2, scheduling_unit=5)
                yield workload.terminate_workload(), workload.ignore_workload_response

            hparams = self.hparams.copy()
            hparams["training_end"] = os.path.join(temp_directory, "training_end.log")

            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class=estimator_xor_model.XORTrialEndOfTrainingHook,
                hparams=hparams,
                workloads=make_workloads(),
                scheduling_unit=5,
            )
            controller.run()

            with open(hparams["training_end"], "r") as fp:
                assert fp.readline() == "success"

    @pytest.mark.parametrize("stop_early,request_stop_step_id", [("train", 1), ("validation", 2)])
    def test_early_stopping(self, stop_early: str, request_stop_step_id: int) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate(request_stop_step_id=request_stop_step_id)
            yield from trainer.send(steps=2, validation_freq=2, scheduling_unit=5)
            tm, vm = trainer.result()
            yield workload.terminate_workload(), workload.ignore_workload_response

        hparams = dict(self.hparams)
        hparams["stop_early"] = stop_early
        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=estimator_xor_model.XORTrial,
            hparams=hparams,
            workloads=make_workloads(),
            scheduling_unit=5,
        )
        controller.run()


class TestLinearTrial:
    def setup_method(self) -> None:
        self.hparams = {
            "learning_rate": 0.0001,
            "global_batch_size": 4,
        }

    def teardown_method(self) -> None:
        # Cleanup leftover environment variable state.
        for key in harness.ENVIRONMENT_VARIABLE_KEYS:
            if key in os.environ:
                del os.environ[key]

    def test_custom_reducer(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            # Test >1 validation to ensure that resetting the allgather_op list is working.
            yield from trainer.send(steps=2, validation_freq=1, scheduling_unit=1)
            training_metrics, validation_metrics = trainer.result()

            label_sum = estimator_linear_model.validation_label_sum()
            for metrics in validation_metrics:
                assert metrics["label_sum_tensor_fn"] == label_sum
                assert metrics["label_sum_tensor_cls"] == label_sum
                assert metrics["label_sum_list_fn"] == 2 * label_sum
                assert metrics["label_sum_list_cls"] == 2 * label_sum
                assert metrics["label_sum_dict_fn"] == 2 * label_sum
                assert metrics["label_sum_dict_cls"] == 2 * label_sum

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=estimator_linear_model.LinearEstimator,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=0,
        )
        controller.run()


def test_create_trial_instance() -> None:
    utils.create_trial_instance(estimator_xor_model.XORTrial)

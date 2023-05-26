# type: ignore
import os
import pathlib
import tempfile
from typing import Any, Dict, List, Optional

import pytest
import tensorflow as tf
from tensorflow.python.training.tracking import tracking

import determined as det
from determined import estimator, workload
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import estimator_linear_model, estimator_xor_model


def xor_trial_controller(
    hparams: Dict[str, Any],
    workloads: workload.Stream,
    scheduling_unit: int = 1,
    exp_config: Optional[Dict] = None,
    checkpoint_dir: Optional[str] = None,
    latest_checkpoint: Optional[Dict[str, Any]] = None,
    steps_completed: int = 0,
) -> det.TrialController:
    return utils.make_trial_controller_from_trial_implementation(
        trial_class=estimator_xor_model.XORTrial,
        hparams=hparams,
        workloads=workloads,
        scheduling_unit=scheduling_unit,
        exp_config=exp_config,
        trial_seed=325,
        checkpoint_dir=checkpoint_dir,
        latest_checkpoint=latest_checkpoint,
        steps_completed=steps_completed,
    )


class TestXORTrial:
    def setup_method(self) -> None:
        self.hparams = {
            "hidden_size": 2,
            "learning_rate": 0.1,
            "global_batch_size": 4,
            "optimizer": "sgd",
            "shuffle": False,
        }

    def test_xor_training(self) -> None:
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

        controller = xor_trial_controller(self.hparams, make_workloads(), scheduling_unit=1000)
        controller.run()

    def test_reproducibility(self) -> None:
        def controller_fn(workloads: workload.Stream) -> det.TrialController:
            return xor_trial_controller(self.hparams, workloads, scheduling_unit=100)

        utils.reproducibility_test(
            controller_fn=controller_fn, steps=3, validation_freq=1, scheduling_unit=100
        )

    def test_checkpointing(self, tmp_path: pathlib.Path) -> None:
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        latest_checkpoint = None
        steps_completed = 0
        old_loss = -1

        def make_workloads_1() -> workload.Stream:
            nonlocal old_loss

            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=1, validation_freq=1, scheduling_unit=10)
            training_metrics, validation_metrics = trainer.result()
            old_loss = validation_metrics[-1]["loss"]

            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint, steps_completed
            latest_checkpoint = interceptor.metrics_result()["uuid"]
            steps_completed = trainer.get_steps_completed()

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

        controller = xor_trial_controller(
            self.hparams,
            make_workloads_2(),
            scheduling_unit=10,
            checkpoint_dir=checkpoint_dir,
            latest_checkpoint=latest_checkpoint,
            steps_completed=steps_completed,
        )
        controller.run()

    def test_checkpointing_with_serving_fn(self, tmp_path: pathlib.Path) -> None:
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        latest_checkpoint = None

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1, scheduling_unit=10)

            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint
            latest_checkpoint = interceptor.metrics_result()["uuid"]

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
        export_path = os.path.join(checkpoint_dir, latest_checkpoint, "inference")
        assert os.path.exists(export_path)
        _, dirs, _ = next(os.walk(export_path))
        assert len(dirs) == 1
        load_saved_model(os.path.join(export_path, dirs[0]))

    def test_optimizer_state(self, tmp_path: pathlib.Path) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream,
            checkpoint_dir: Optional[str] = None,
            latest_checkpoint: Optional[Dict[str, Any]] = None,
            steps_completed: int = 0,
        ) -> det.TrialController:
            hparams = {**self.hparams, "optimizer": "adam"}
            return xor_trial_controller(
                hparams,
                workloads,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
                steps_completed=steps_completed,
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

    def test_custom_hook(self, tmp_path: pathlib.Path) -> None:
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        latest_checkpoint = None
        steps_completed = 0

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=5, scheduling_unit=5)

            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint, steps_completed
            latest_checkpoint = interceptor.metrics_result()["uuid"]
            steps_completed = trainer.get_steps_completed()

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
        verify_callback(os.path.join(checkpoint_dir, latest_checkpoint), checkpoint_num=1)

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=estimator_xor_model.XORTrialWithCustomHook,
            hparams=self.hparams,
            workloads=make_workloads(),
            scheduling_unit=5,
            checkpoint_dir=checkpoint_dir,
            latest_checkpoint=latest_checkpoint,
            steps_completed=steps_completed,
        )
        controller.run()
        verify_callback(os.path.join(checkpoint_dir, latest_checkpoint), checkpoint_num=2)

    def test_end_of_training_hook(self):
        with tempfile.TemporaryDirectory() as temp_directory:

            def make_workloads() -> workload.Stream:
                trainer = utils.TrainAndValidate()

                yield from trainer.send(steps=2, validation_freq=2, scheduling_unit=5)

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

    def test_require_global_batch_size(self) -> None:
        utils.ensure_requires_global_batch_size(
            estimator_linear_model.LinearEstimator, self.hparams
        )

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

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=estimator_linear_model.LinearEstimator,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=0,
        )
        controller.run()


@pytest.mark.parametrize("ckpt_ver", ["0.17.6", "0.17.7"])
def test_checkpoint_loading(ckpt_ver):
    checkpoint_dir = os.path.join(
        utils.fixtures_path("ancient-checkpoints"), f"{ckpt_ver}-estimator"
    )
    estm = estimator.load_estimator_from_checkpoint_path(checkpoint_dir)
    assert isinstance(estm, tracking.AutoTrackable), type(estm)


@pytest.mark.tf1_cpu
def test_rng_restore():
    def make_checkpoint() -> workload.Stream:
        trainer = utils.TrainAndValidate()

        yield from trainer.send(steps=1, validation_freq=1, scheduling_unit=1)

    def make_workloads_with_metrics(metrics_storage: List) -> workload.Stream:
        trainer = utils.TrainAndValidate()

        yield from trainer.send(steps=5, validation_freq=1, scheduling_unit=1)
        _, validation_metrics = trainer.result()

        metrics_storage += validation_metrics

    config_base = utils.load_config(utils.fixtures_path("estimator_no_op/const.yaml"))
    hparams = config_base["hyperparameters"]

    example_path = utils.fixtures_path("estimator_no_op/model_def.py")
    trial_class = utils.import_class_from_module("NoopEstimator", example_path)
    trial_class._searcher_metric = "validation_error"

    trial_B_metrics = []
    trial_C_metrics = []

    trial_A_controller = utils.make_trial_controller_from_trial_implementation(
        trial_class=trial_class, hparams=hparams, workloads=make_checkpoint(), trial_seed=325
    )

    trial_A_controller.run()

    # copy checkpoint
    checkpoint_dir = trial_A_controller.estimator_dir

    # reset random seed after checkpointing
    trial_A_controller.set_random_seed(0)

    trial_B_controller = utils.make_trial_controller_from_trial_implementation(
        trial_class=trial_class,
        hparams=hparams,
        workloads=make_workloads_with_metrics(trial_B_metrics),
        latest_checkpoint=str(checkpoint_dir),
        steps_completed=1,
    )

    trial_B_controller.run()

    # reset random seed before rerun
    trial_B_controller.set_random_seed(1)

    trial_C_controller = utils.make_trial_controller_from_trial_implementation(
        trial_class=trial_class,
        hparams=hparams,
        workloads=make_workloads_with_metrics(trial_C_metrics),
        latest_checkpoint=str(checkpoint_dir),
        steps_completed=1,
    )

    trial_C_controller.run()

    assert len(trial_B_metrics) == len(trial_C_metrics) == 5
    assert trial_B_metrics == trial_C_metrics

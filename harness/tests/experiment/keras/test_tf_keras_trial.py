import subprocess
from pathlib import Path
from typing import Any, Callable, Dict, Optional

import pytest
import tensorflow as tf
from packaging import version

import determined as det
from determined import workload
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import tf_keras_one_var_model, tf_keras_xor_model  # noqa: I100


def test_executing_eagerly():
    """
    While this may seem like a test we do not need, we actually do. This
    essentially tests that there are no side effects to our imports that would
    cause eager execution to be turned off.
    """
    is_tf2 = version.parse(tf.__version__) >= version.parse("2.0.0")  # type: bool
    if is_tf2:
        assert tf.executing_eagerly()


@pytest.fixture(
    scope="function",
    params=[
        tf_keras_xor_model.XORTrial,
        tf_keras_xor_model.XORTrialOldOptimizerAPI,
        tf_keras_xor_model.XORTrialWithTrainingMetrics,
        tf_keras_xor_model.XORTrialWithCustomObjects,
        tf_keras_xor_model.XORTrialWithDataLayer,
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
        trial_seed: int = 0,
        checkpoint_dir: Optional[str] = None,
        latest_checkpoint: Optional[Dict[str, Any]] = None,
    ) -> det.TrialController:
        return utils.make_trial_controller_from_trial_implementation(
            request.param,
            hparams,
            workloads,
            scheduling_unit=scheduling_unit,
            trial_seed=trial_seed,
            checkpoint_dir=checkpoint_dir,
            latest_checkpoint=latest_checkpoint,
        )

    return _xor_trial_controller


class TestKerasTrial:
    def setup_method(self) -> None:
        # This training setup is not guaranteed to converge in general,
        # but has been tested with this random seed.  If changing this
        # random seed, verify the initial conditions converge.
        self.trial_seed = 325
        self.hparams = {
            "hidden_size": 2,
            "learning_rate": 0.1,
            "global_batch_size": 4,
            "trial_type": "default",
        }

    def teardown_method(self) -> None:
        # Reset the default graph state after each invocation so tests can
        # fully rely on the graph-level seed for determinism.
        tf.compat.v1.reset_default_graph()

    # The following unit tests are run with a specific trial implementation.

    def test_xor_training_with_metrics(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10)
            training_metrics, validation_metrics = trainer.result()

            for metrics in training_metrics:
                assert "categorical_accuracy" in metrics
                assert "predictions" in metrics

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            tf_keras_xor_model.XORTrialWithTrainingMetrics,
            self.hparams,
            make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    @pytest.mark.parametrize("test_checkpointing", [False, True])
    def test_one_var_training(self, test_checkpointing, tmp_path):
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        latest_checkpoint = None

        # In the test_checkpointing case, we will call make_workloads() twice but batches and w
        # will persist across both calls.
        batches = enumerate([[0, 1, 2], [3, 4, 5], [6, 7, 8], [9]])
        w = 0.0

        trial_class = tf_keras_one_var_model.OneVarTrial

        def make_workloads() -> workload.Stream:
            nonlocal w
            interceptor = workload.WorkloadResponseInterceptor()

            for idx, batch in batches:
                yield from interceptor.send(workload.train_workload(1))
                metrics = interceptor.metrics_result()

                # Calculate what the loss should be.
                loss = trial_class.calc_loss(w, batch)

                epsilon = 0.0001
                assert abs(metrics["metrics"]["avg_metrics"]["loss"] - loss) < epsilon

                # Update what the weight should be.
                w = w - hparams["learning_rate"] * trial_class.calc_gradient(w, batch)

                if test_checkpointing and idx == 3:
                    # Checkpoint and let the next TrialController finish the work.l
                    interceptor = workload.WorkloadResponseInterceptor()
                    yield from interceptor.send(workload.checkpoint_workload())
                    nonlocal latest_checkpoint
                    latest_checkpoint = interceptor.metrics_result()["metrics"].__json__()
                    break

            yield workload.terminate_workload(), workload.ignore_workload_response

        hparams = {"learning_rate": 0.001, "global_batch_size": 3, "dataset_range": 10}
        exp_config = utils.make_default_exp_config(
            hparams, scheduling_unit=100, searcher_metric=trial_class._searcher_metric
        )
        exp_config["records_per_epoch"] = 100
        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class,
            hparams,
            make_workloads(),
            exp_config=exp_config,
            trial_seed=self.trial_seed,
            checkpoint_dir=checkpoint_dir,
        )
        controller.run()

        # In the checkpointing case, we need to create another controller to finish training.
        if test_checkpointing:
            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class,
                hparams,
                make_workloads(),
                exp_config=exp_config,
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
            )
            controller.run()

    # The following unit tests are generally applicable and run on the cross
    # product of all implementations.

    def test_xor_training(self, xor_trial_controller: Callable) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=1, scheduling_unit=100)
            training_metrics, validation_metrics = trainer.result()

            # We expect the validation error and training loss to be
            # monotonically decreasing.

            # TODO(DET-1597): actually use a model and optimizer where the losses
            # are monotonically decreasing.
            for older, newer in zip(training_metrics[::100], training_metrics[::100][1:]):
                assert newer["loss"] <= older["loss"]

            for older, newer in zip(validation_metrics, validation_metrics[1:]):
                assert newer["val_categorical_error"] <= older["val_categorical_error"]

            epsilon = 0.0001
            assert abs(validation_metrics[-1]["val_categorical_error"]) < epsilon

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = xor_trial_controller(
            self.hparams, make_workloads(), scheduling_unit=100, trial_seed=self.trial_seed
        )
        controller.run()

    def test_checkpointing(self, tmp_path: Path, xor_trial_controller: Callable) -> None:
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        latest_checkpoint = None
        old_loss = -1

        def make_workloads_1() -> workload.Stream:
            nonlocal old_loss

            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10)
            training_metrics, validation_metrics = trainer.result()
            old_loss = validation_metrics[-1]["val_loss"]

            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint
            latest_checkpoint = interceptor.metrics_result()["metrics"].__json__()

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = xor_trial_controller(
            self.hparams,
            make_workloads_1(),
            trial_seed=self.trial_seed,
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

            new_loss = metrics["metrics"]["validation_metrics"]["val_loss"]
            assert new_loss == pytest.approx(old_loss)

            yield workload.terminate_workload(), workload.ignore_workload_response

        controller = xor_trial_controller(
            self.hparams,
            make_workloads_2(),
            trial_seed=self.trial_seed,
            checkpoint_dir=checkpoint_dir,
            latest_checkpoint=latest_checkpoint,
        )
        controller.run()

    def test_optimizer_state(self, tmp_path: Path, xor_trial_controller: Callable) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream,
            checkpoint_dir: Optional[str] = None,
            latest_checkpoint: Optional[Dict[str, Any]] = None,
        ) -> det.TrialController:
            return xor_trial_controller(
                self.hparams,
                workloads,
                scheduling_unit=100,
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
            )

        utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

    def test_reproducibility(self, xor_trial_controller: Callable) -> None:
        def controller_fn(workloads: workload.Stream) -> det.TrialController:
            return xor_trial_controller(
                self.hparams, workloads, scheduling_unit=100, trial_seed=self.trial_seed
            )

        utils.reproducibility_test(
            controller_fn=controller_fn, steps=3, validation_freq=1, scheduling_unit=100
        )

    def test_early_stopping(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate(request_stop_step_id=1)
            yield from trainer.send(steps=100, validation_freq=2, scheduling_unit=5)
            tm, vm = trainer.result()
            yield workload.terminate_workload(), workload.ignore_workload_response

        hparams = dict(self.hparams)
        hparams["stop_early"] = True

        controller = utils.make_trial_controller_from_trial_implementation(
            tf_keras_xor_model.XORTrial,
            hparams,
            make_workloads(),
            scheduling_unit=5,
        )
        controller.run()

    def test_callbacks(self):
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=15, validation_freq=4, scheduling_unit=5)
            training_metrics, validation_metrics = trainer.result()

            yield workload.terminate_workload(), workload.ignore_workload_response

        hparams = {
            "learning_rate": 0.001,
            "global_batch_size": 3,
            "dataset_range": 10,
            # 15 steps * 5 batches per step * 3 records per batch // 12 records per epoch
            "epochs": 15 * 5 * 3 // 12,
            # steps // validation_freq
            "validations": 3,
        }
        exp_config = utils.make_default_exp_config(
            hparams, scheduling_unit=100, searcher_metric="val_loss"
        )
        exp_config["records_per_epoch"] = 12

        controller = utils.make_trial_controller_from_trial_implementation(
            tf_keras_one_var_model.OneVarTrial,
            hparams,
            make_workloads(),
            exp_config=exp_config,
        )
        controller.run()


def test_surface_native_error():
    cmd = ["python3", utils.fixtures_path("tf_keras_runtime_error.py")]
    with subprocess.Popen(cmd, stderr=subprocess.PIPE) as p:
        err = p.stderr.read()
        assert p.wait() != 0
        if tf.executing_eagerly():
            assert (
                b"ValueError: Shapes (None, 10) and (None, 1) are incompatible" in err
                or b"ValueError: Input 0 of layer sequential is incompatible with the "
                b"layer: : expected min_ndim=2, found ndim=1. Full shape received: [1]" in err
                or b"ValueError: Input 0 of layer sequential is incompatible with the "
                b"layer: : expected min_ndim=2, found ndim=1. Full shape received: (1,)" in err
            ), err.decode("utf8")
        else:
            assert (
                b"ValueError: Input 0 of layer sequential is incompatible with the layer" in err
            ), err.decode("utf8")


def test_create_trial_instance() -> None:
    utils.create_trial_instance(tf_keras_xor_model.XORTrial)

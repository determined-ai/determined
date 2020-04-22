import os
import subprocess
from pathlib import Path
from typing import Any, Callable, Dict, Optional

import pytest
import simplejson
import tensorflow as tf
from packaging import version

import determined as det
from determined import workload
from determined.exec import harness
from tests.unit.experiment import utils  # noqa: I100
from tests.unit.experiment.fixtures import tf_keras_one_var_model, tf_keras_xor_model  # noqa: I100


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
        tf_keras_xor_model.XORTrialWithTrainingMetrics,
        tf_keras_xor_model.XORTrialWithCustomObjects,
        tf_keras_xor_model.XORTrialWithDataLayer,
        utils.fixtures_path("tf_keras_xor_model_native.py"),
    ],
)
def xor_trial_controller(request):
    """
    This fixture will provide a function that takes a hyperparameters
    dictionary as input and returns a trial controller. It is parameterized
    over different implementations (both native and trial), so that any test
    that uses it may test a full set of implementations.
    """
    if isinstance(request.param, str):

        def _xor_trial_controller(
            hparams: Dict[str, Any],
            workloads: workload.Stream,
            batches_per_step: int = 1,
            load_path: Optional[str] = None,
            trial_seed: int = 0,
        ) -> det.TrialController:
            return utils.make_trial_controller_from_native_implementation(
                command=request.param,
                hparams=hparams,
                workloads=workloads,
                batches_per_step=batches_per_step,
                load_path=load_path,
                trial_seed=trial_seed,
            )

        return _xor_trial_controller
    else:

        def _xor_trial_controller(
            hparams: Dict[str, Any],
            workloads: workload.Stream,
            batches_per_step: int = 1,
            load_path: Optional[str] = None,
            trial_seed: int = 0,
        ) -> det.TrialController:
            return utils.make_trial_controller_from_trial_implementation(
                request.param,
                hparams,
                workloads,
                batches_per_step=batches_per_step,
                load_path=load_path,
                trial_seed=trial_seed,
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
        os.environ["DET_HPARAMS"] = simplejson.dumps(self.hparams)

    def teardown_method(self) -> None:
        # Reset the default graph state after each invocation so tests can
        # fully rely on the graph-level seed for determinism.
        tf.compat.v1.reset_default_graph()

        # Cleanup leftover environment variable state.
        for key in harness.ENVIRONMENT_VARIABLE_KEYS:
            if key in os.environ:
                del os.environ[key]

    # The following unit tests are run with a specific trial implementation.

    def test_xor_training_with_metrics(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10)
            training_metrics, validation_metrics = trainer.result()

            for metrics in training_metrics:
                assert "categorical_accuracy" in metrics
                assert "predictions" in metrics

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            tf_keras_xor_model.XORTrialWithTrainingMetrics,
            self.hparams,
            make_workloads(),
            trial_seed=self.trial_seed,
        )
        controller.run()

    @pytest.mark.parametrize("test_checkpointing", [False, True])
    def test_one_var_training(self, test_checkpointing, tmp_path):
        checkpoint_dir = tmp_path.joinpath("checkpoint")

        # In the test_checkpointing case, we will call make_workloads() twice but batches and w
        # will persist across both calls.
        batches = enumerate([[0, 1, 2], [3, 4, 5], [6, 7, 8], [9]])
        w = 0.0

        trial_class = tf_keras_one_var_model.OneVarTrial

        def make_workloads() -> workload.Stream:
            nonlocal w
            interceptor = workload.WorkloadResponseInterceptor()

            for idx, batch in batches:
                yield from interceptor.send(workload.train_workload(1), [1])
                metrics = interceptor.metrics_result()

                # Calculate what the loss should be.
                loss = trial_class.calc_loss(w, batch)

                assert metrics["avg_metrics"]["loss"] == pytest.approx(loss)

                # Update what the weight should be.
                w = w - hparams["learning_rate"] * trial_class.calc_gradient(w, batch)

                if test_checkpointing and idx == 3:
                    # Checkpoint and let the next TrialController finish the work.l
                    yield workload.checkpoint_workload(), [
                        checkpoint_dir
                    ], workload.ignore_workload_response
                    break

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        hparams = {"learning_rate": 0.001, "global_batch_size": 3, "dataset_range": 10}
        # TODO(DET-2436): Add a unit test for native implementation with tf dataset.
        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class, hparams, make_workloads(), trial_seed=self.trial_seed,
        )
        controller.run()

        # In the checkpointing case, we need to create another controller to finish training.
        if test_checkpointing:
            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class,
                hparams,
                make_workloads(),
                load_path=checkpoint_dir,
                trial_seed=self.trial_seed,
            )
            controller.run()

    # The following unit tests are generally applicable and run on the cross
    # product of all implementations.

    def test_xor_training(self, xor_trial_controller: Callable) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=1, batches_per_step=100)
            training_metrics, validation_metrics = trainer.result()

            # We expect the validation error and training loss to be
            # monotonically decreasing.

            # TODO(DET-1597): actually use a model and optimizer where the losses
            # are monotonically decreasing.
            for older, newer in zip(training_metrics[::100], training_metrics[::100][1:]):
                assert newer["loss"] <= older["loss"]

            for older, newer in zip(validation_metrics, validation_metrics[1:]):
                assert newer["val_categorical_error"] <= older["val_categorical_error"]

            assert validation_metrics[-1]["val_categorical_error"] == pytest.approx(0.0)

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = xor_trial_controller(
            self.hparams, make_workloads(), batches_per_step=100, trial_seed=self.trial_seed
        )
        controller.run()

    def test_checkpointing(self, tmp_path: Path, xor_trial_controller: Callable) -> None:
        checkpoint_dir = tmp_path.joinpath("checkpoint")
        old_loss = -1

        def make_workloads_1() -> workload.Stream:
            nonlocal old_loss

            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10)
            training_metrics, validation_metrics = trainer.result()
            old_loss = validation_metrics[-1]["val_loss"]

            yield workload.checkpoint_workload(), [
                checkpoint_dir
            ], workload.ignore_workload_response

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = xor_trial_controller(
            self.hparams, make_workloads_1(), trial_seed=self.trial_seed
        )
        controller.run()

        # Restore the checkpoint on a new trial instance and recompute
        # validation. The validation error should be the same as it was
        # previously.
        def make_workloads_2() -> workload.Stream:
            interceptor = workload.WorkloadResponseInterceptor()

            yield from interceptor.send(workload.validation_workload(), [])
            metrics = interceptor.metrics_result()

            new_loss = metrics["validation_metrics"]["val_loss"]
            assert new_loss == pytest.approx(old_loss)

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = xor_trial_controller(
            self.hparams, make_workloads_2(), load_path=checkpoint_dir, trial_seed=self.trial_seed
        )
        controller.run()

    def test_optimizer_state(self, tmp_path: Path, xor_trial_controller: Callable) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream, load_path: Optional[str] = None
        ) -> det.TrialController:
            return xor_trial_controller(
                self.hparams,
                workloads,
                batches_per_step=100,
                load_path=load_path,
                trial_seed=self.trial_seed,
            )

        utils.optimizer_state_test(make_trial_controller_fn, tmp_path)

    def test_reproducibility(self, xor_trial_controller: Callable) -> None:
        def controller_fn(workloads: workload.Stream) -> det.TrialController:
            return xor_trial_controller(
                self.hparams, workloads, batches_per_step=100, trial_seed=self.trial_seed
            )

        utils.reproducibility_test(
            controller_fn=controller_fn, steps=3, validation_freq=1, batches_per_step=100,
        )


def test_surface_native_error():
    cmd = ["python3", utils.fixtures_path("tf_keras_runtime_error.py")]
    with subprocess.Popen(cmd, stderr=subprocess.PIPE) as p:
        err = p.stderr.read()
        assert p.wait() != 0
        if tf.executing_eagerly():
            assert b"ValueError: Shapes (None, 10) and (None, 1) are incompatible" in err
        else:
            assert b"ValueError: Input 0 of layer sequential is incompatible with the layer" in err


def test_local_mode() -> None:
    utils.run_local_mode(utils.fixtures_path("tf_keras_xor_model_native.py"))


def test_create_trial_instance() -> None:
    utils.create_trial_instance(tf_keras_xor_model.XORTrial)

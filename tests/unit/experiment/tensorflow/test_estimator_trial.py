import os
from pathlib import Path
from typing import Any, Callable, Dict, Optional

import pytest
import tensorflow as tf

import determined as det
from determined import workload
from determined.exec import harness
from tests.unit.experiment import utils  # noqa: I100
from tests.unit.experiment.fixtures import estimator_xor_model


@pytest.fixture(
    scope="function",
    params=[
        estimator_xor_model.XORTrial,
        estimator_xor_model.XORTrialDataLayer,
        utils.fixtures_path("estimator_xor_model_native.py"),
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
        ) -> det.TrialController:
            return utils.make_trial_controller_from_native_implementation(
                command=request.param,
                hparams=hparams,
                workloads=workloads,
                batches_per_step=batches_per_step,
                load_path=load_path,
                trial_seed=325,
            )

        return _xor_trial_controller
    else:

        def _xor_trial_controller(
            hparams: Dict[str, Any],
            workloads: workload.Stream,
            batches_per_step: int = 1,
            load_path: Optional[str] = None,
            exp_config: Optional[Dict] = None,
        ) -> det.TrialController:
            if request.param == estimator_xor_model.XORTrialDataLayer:
                exp_config = utils.make_default_exp_config(
                    hparams=hparams, batches_per_step=batches_per_step,
                )
                exp_config["data"] = exp_config.get("data", {})
                exp_config["data"]["skip_checkpointing_input"] = True

            return utils.make_trial_controller_from_trial_implementation(
                trial_class=request.param,
                hparams=hparams,
                workloads=workloads,
                batches_per_step=batches_per_step,
                load_path=load_path,
                exp_config=exp_config,
            )

        return _xor_trial_controller


class TestXORTrial:
    def setup_method(self) -> None:
        os.environ["DET_RENDEZVOUS_INFO"] = '{"rank": 0, "addrs": ["localhost"]}'
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

            yield from trainer.send(steps=10, validation_freq=5, batches_per_step=1000)
            training_metrics, validation_metrics = trainer.result()

            # We expect the training loss to be monotonically decreasing and the
            # accuracy to be monotonically increasing.
            for older, newer in zip(training_metrics, training_metrics[1:]):
                assert newer["loss"] < older["loss"]

            for older, newer in zip(validation_metrics, validation_metrics[1:]):
                assert newer["accuracy"] >= older["accuracy"]

            # The final accuracy should be 100%.
            assert validation_metrics[-1]["accuracy"] == pytest.approx(1.0)

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = xor_trial_controller(self.hparams, make_workloads(), batches_per_step=1000)
        controller.run()

    def test_reproducibility(self, xor_trial_controller: Callable) -> None:
        def controller_fn(workloads: workload.Stream) -> det.TrialController:
            return xor_trial_controller(self.hparams, workloads, batches_per_step=100)

        utils.reproducibility_test(
            controller_fn=controller_fn, steps=3, validation_freq=1, batches_per_step=100,
        )

    def test_checkpointing(self, tmp_path: Path, xor_trial_controller: Callable) -> None:
        checkpoint_dir = tmp_path.joinpath("checkpoint")
        old_loss = -1

        def make_workloads_1() -> workload.Stream:
            nonlocal old_loss

            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=1, validation_freq=1, batches_per_step=10)
            training_metrics, validation_metrics = trainer.result()
            old_loss = validation_metrics[-1]["loss"]

            yield workload.checkpoint_workload(), [
                checkpoint_dir
            ], workload.ignore_workload_response

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = xor_trial_controller(self.hparams, make_workloads_1(), batches_per_step=10)
        controller.run()

        # Restore the checkpoint on a new trial instance and recompute
        # validation. The validation error should be the same as it was
        # previously.
        def make_workloads_2() -> workload.Stream:
            interceptor = workload.WorkloadResponseInterceptor()

            yield from interceptor.send(workload.validation_workload(), [])
            metrics = interceptor.metrics_result()

            new_loss = metrics["validation_metrics"]["loss"]
            assert new_loss == pytest.approx(old_loss)

            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = xor_trial_controller(
            self.hparams, make_workloads_2(), batches_per_step=10, load_path=checkpoint_dir
        )
        controller.run()

    def test_checkpointing_with_serving_fn(
        self, tmp_path: Path, xor_trial_controller: Callable
    ) -> None:
        checkpoint_dir = tmp_path.joinpath("checkpoint")

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(steps=1, validation_freq=1, batches_per_step=10)
            yield workload.checkpoint_workload(), [
                checkpoint_dir
            ], workload.ignore_workload_response
            yield workload.terminate_workload(), [], workload.ignore_workload_response

        controller = xor_trial_controller(self.hparams, make_workloads(), batches_per_step=10)
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
        export_path = os.path.join(checkpoint_dir, "inference")
        assert os.path.exists(export_path)
        _, dirs, _ = next(os.walk(export_path))
        assert len(dirs) == 1
        load_saved_model(os.path.join(export_path, dirs[0]))

    def test_optimizer_state(self, tmp_path: Path, xor_trial_controller: Callable) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream, load_path: Optional[str] = None
        ) -> det.TrialController:
            hparams = {**self.hparams, "optimizer": "adam"}
            return xor_trial_controller(hparams, workloads, load_path=load_path)

        utils.optimizer_state_test(make_trial_controller_fn, tmp_path)


def test_local_mode() -> None:
    utils.run_local_mode(utils.fixtures_path("estimator_xor_model_native.py"))


def test_create_trial_instance() -> None:
    utils.create_trial_instance(estimator_xor_model.XORTrial)

# type: ignore
import copy
import json
import os
import pathlib
from typing import Any, Dict, Iterator, Optional

import pytest
import torch
from deepspeed.runtime import config_utils

import determined
import determined.pytorch.deepspeed as det_deepspeed
from determined import workload
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import deepspeed_linear_model

ds_config_path = str(
    pathlib.Path(__file__).resolve().parent.parent.joinpath("fixtures/ds_config.json")
)
deepspeed_config = json.load(
    open(ds_config_path, "r"),
    object_pairs_hook=config_utils.dict_raise_error_on_duplicate_keys,
)


@pytest.fixture
def manual_init_distributed() -> Iterator[None]:
    """Set DET_MANUAL_INIT_DISTRIBUTED in os.environ for the duration of a single test."""

    os.environ["DET_MANUAL_INIT_DISTRIBUTED"] = "1"
    try:
        yield
    finally:
        del os.environ["DET_MANUAL_INIT_DISTRIBUTED"]


@pytest.mark.skipif(not torch.cuda.is_available(), reason="no gpu available")
@pytest.mark.deepspeed
@pytest.mark.gpu
class TestDeepSpeedTrial:
    def setup_method(self) -> None:
        # These environment variables are usually set by the launcher, but we set them manually here
        # since they are required internally by the deepspeed model engine.
        os.environ["RANK"] = "0"
        os.environ["LOCAL_RANK"] = "0"
        os.environ["WORLD_SIZE"] = "1"
        os.environ["MASTER_ADDR"] = "localhost"
        os.environ["MASTER_PORT"] = "29500"

        self.trial_seed = 17
        self.hparams = {
            "global_batch_size": 16,
            "deepspeed_config": deepspeed_config,
            "test_manual_init_distributed": False,
            "test_fail_manual_init_distributed": False,
            "test_manual_dataloader": False,
            "test_fail_dataset_repro_check": False,
            "test_manual_grad_acc": False,
            "test_fail_manual_grad_acc": False,
            "return_non_scalar_metrics": False,
            "test_custom_reducer": False,
        }
        self.data_parallel_only_auto_train_batch_calls = (
            deepspeed_config["train_batch_size"]
            // deepspeed_config["train_micro_batch_size_per_gpu"]
        )

    def teardown_method(self) -> None:
        # Remove set environment variables
        for key in ["RANK", "LOCAL_RANK", "WORLD_SIZE", "MASTER_ADDR", "MASTER_PORT"]:
            del os.environ[key]

    def test_fail_manual_init_distributed(self, manual_init_distributed: None):
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_fail_manual_init_distributed"] = True

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=10,
                validation_freq=10,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "loss" in metrics

        with pytest.raises(AssertionError, match=r"Distributed backend is not initialized. .*"):
            _ = utils.make_trial_controller_from_trial_implementation(
                trial_class=deepspeed_linear_model.LinearDeepSpeedTrial,
                hparams=updated_hparams,
                workloads=make_workloads(),
                trial_seed=self.trial_seed,
                expose_gpus=True,
            )

    def test_manual_init_distributed(self, manual_init_distributed: None):
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_manual_init_distributed"] = True

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=10,
                validation_freq=10,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "loss" in metrics

        _ = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearDeepSpeedTrial,
            hparams=updated_hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
            expose_gpus=True,
        )
        assert torch.distributed.is_initialized()

    def test_linear_model(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=10,
                validation_freq=10,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "loss" in metrics

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearDeepSpeedTrial,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
            expose_gpus=True,
        )
        controller.run()

    def test_manual_grad_acc_metrics(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_manual_grad_acc"] = True

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10, train_batch_calls=1)
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "loss" in metrics

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearDeepSpeedTrial,
            hparams=updated_hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
            expose_gpus=True,
        )
        controller.run()

    def test_fail_manual_grad_acc_metrics(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_fail_manual_grad_acc"] = True

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=10, validation_freq=10, train_batch_calls=1)
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "loss" in metrics

        with pytest.raises(AssertionError, match="did not train for gradient accumulation steps"):
            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class=deepspeed_linear_model.LinearDeepSpeedTrial,
                hparams=updated_hparams,
                workloads=make_workloads(),
                trial_seed=self.trial_seed,
                expose_gpus=True,
            )
            controller.run()

    def test_custom_dataloader(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_manual_dataloader"] = True

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=10,
                validation_freq=10,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "loss" in metrics

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearDeepSpeedTrial,
            hparams=updated_hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
            expose_gpus=True,
        )
        controller.run()

    def test_fail_dataset_repro_check(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_fail_dataset_repro_check"] = True

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=10,
                validation_freq=10,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "loss" in metrics

        with pytest.raises(RuntimeError, match=r".* reproducibility .* disable this check .*"):
            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class=deepspeed_linear_model.LinearDeepSpeedTrial,
                hparams=updated_hparams,
                workloads=make_workloads(),
                trial_seed=self.trial_seed,
                expose_gpus=True,
            )
            controller.run()

    def test_invalid_valid_dataset(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=10,
                validation_freq=10,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )

        with pytest.raises(
            determined.errors.InvalidExperimentException,
            match=r".* train micro batches .* should not be less than .*",
        ):
            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class=deepspeed_linear_model.InvalidValidDatasetTrial,
                hparams=self.hparams,
                workloads=make_workloads(),
                trial_seed=self.trial_seed,
                expose_gpus=True,
            )
            controller.run()

    def test_invalid_train_metric(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=10,
                validation_freq=10,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )

        with pytest.raises(
            determined.errors.InvalidExperimentException,
            match=r"train_batch() must return a dictionary .*",
        ):
            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class=deepspeed_linear_model.InvalidTrainMetricTrial,
                hparams=self.hparams,
                workloads=make_workloads(),
                trial_seed=self.trial_seed,
                expose_gpus=True,
            )
            controller.run()

    def test_invalid_valid_metric(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=10,
                validation_freq=10,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )

        with pytest.raises(
            determined.errors.InvalidExperimentException,
            match=r"evaluate_batch must return a dictionary .*",
        ):
            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class=deepspeed_linear_model.InvalidValidMetricTrial,
                hparams=self.hparams,
                workloads=make_workloads(),
                trial_seed=self.trial_seed,
                expose_gpus=True,
            )
            controller.run()

    def test_differing_valid_metric_keys(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=10,
                validation_freq=10,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )

        with pytest.raises(
            determined.errors.InvalidExperimentException,
            match=r".* metric names must match across all batches .*",
        ):
            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class=deepspeed_linear_model.DifferingValidMetricKeyTrial,
                hparams=self.hparams,
                workloads=make_workloads(),
                trial_seed=self.trial_seed,
                expose_gpus=True,
            )
            controller.run()

    def test_fail_multiple_set_mpu(self):
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=1,
                validation_freq=1,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )

        with pytest.raises(
            determined.errors.InvalidExperimentException, match=r"Only one MPU can be passed .*"
        ):
            controller = utils.make_trial_controller_from_trial_implementation(
                trial_class=deepspeed_linear_model.LinearDeepSpeedTrial,
                hparams=self.hparams,
                workloads=make_workloads(),
                trial_seed=self.trial_seed,
                expose_gpus=True,
            )
            controller.context.set_mpu(
                det_deepspeed.make_data_parallel_mpu(controller.context.distributed)
            )
            controller.context.set_mpu(
                det_deepspeed.make_data_parallel_mpu(controller.context.distributed)
            )

    def test_custom_reducer(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_custom_reducer"] = True

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=10,
                validation_freq=10,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "loss" in metrics

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearDeepSpeedTrial,
            hparams=updated_hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
            expose_gpus=True,
        )
        controller.run()

    def test_linear_non_scalar_metrics(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["return_non_scalar_metrics"] = True

        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=10,
                validation_freq=10,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "loss" in metrics

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearDeepSpeedTrial,
            hparams=updated_hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
            expose_gpus=True,
        )
        controller.run()

    def test_linear_pipeline_model(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(steps=1, validation_freq=1, train_batch_calls=1)
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "loss" in metrics

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearPipelineEngineTrial,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
            expose_gpus=True,
        )
        controller.run()

    def test_two_model_engines(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()

            yield from trainer.send(
                steps=1,
                validation_freq=1,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )
            training_metrics, validation_metrics = trainer.result()

            for metrics in validation_metrics:
                assert "loss1" in metrics
                assert "loss2" in metrics

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearTwoEngineTrial,
            hparams=self.hparams,
            workloads=make_workloads(),
            trial_seed=self.trial_seed,
            expose_gpus=True,
        )
        controller.run()

    def test_checkpointing_and_restoring(self, tmp_path: pathlib.Path) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream,
            checkpoint_dir: Optional[str] = None,
            latest_checkpoint: Optional[Dict[str, Any]] = None,
            steps_completed: int = 0,
        ) -> determined.TrialController:
            return utils.make_trial_controller_from_trial_implementation(
                trial_class=deepspeed_linear_model.LinearPipelineEngineTrial,
                hparams=self.hparams,
                workloads=workloads,
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
                steps_completed=steps_completed,
                expose_gpus=True,
            )

        utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

    def test_restore_invalid_checkpoint(self, tmp_path: pathlib.Path) -> None:
        # Build, train, and save a checkpoint with the normal hyperparameters.
        checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
        latest_checkpoint = None
        steps_completed = 0

        def make_workloads_1() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(
                steps=1,
                validation_freq=1,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )
            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint, steps_completed
            latest_checkpoint = interceptor.metrics_result()["uuid"]
            steps_completed = trainer.get_steps_completed()

        controller1 = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearDeepSpeedTrial,
            hparams=self.hparams,
            workloads=make_workloads_1(),
            trial_seed=self.trial_seed,
            checkpoint_dir=checkpoint_dir,
            expose_gpus=True,
        )
        controller1.run()

        # Verify that an invalid architecture fails to load from the checkpoint.
        def make_workloads_2() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            yield from trainer.send(
                steps=1,
                validation_freq=1,
                train_batch_calls=self.data_parallel_only_auto_train_batch_calls,
            )

        with pytest.raises(AssertionError, match="Failed to load deepspeed checkpoint."):
            controller2 = utils.make_trial_controller_from_trial_implementation(
                trial_class=deepspeed_linear_model.LinearTwoEngineTrial,
                hparams=self.hparams,
                workloads=make_workloads_2(),
                trial_seed=self.trial_seed,
                checkpoint_dir=checkpoint_dir,
                latest_checkpoint=latest_checkpoint,
                steps_completed=steps_completed,
                expose_gpus=True,
            )
            controller2.run()

    def test_reproducibility(self) -> None:
        def controller_fn(workloads: workload.Stream) -> determined.TrialController:
            return utils.make_trial_controller_from_trial_implementation(
                trial_class=deepspeed_linear_model.LinearPipelineEngineTrial,
                hparams=self.hparams,
                workloads=workloads,
                trial_seed=self.trial_seed,
                expose_gpus=True,
            )

        utils.reproducibility_test(controller_fn, steps=1000, validation_freq=100)

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

        hparams1 = dict(self.hparams)
        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearCallbackTrial,
            hparams=hparams1,
            workloads=make_workloads1(),
            checkpoint_dir=str(checkpoint_dir),
            expose_gpus=True,
        )
        controller.run()
        assert controller.trial.counter.trial_shutdowns == 1

        # Verify the checkpoint loading callback works.
        def make_workloads2() -> workload.Stream:
            yield workload.train_workload(1, 1, 0, 2), workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearCallbackTrial,
            hparams=self.hparams,
            workloads=make_workloads2(),
            checkpoint_dir=str(checkpoint_dir),
            latest_checkpoint=latest_checkpoint,
            steps_completed=steps_completed,
            expose_gpus=True,
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


@pytest.mark.deepspeed
def test_overwrite_deepspeed_config() -> None:
    base_ds_config = deepspeed_config
    source_ds_config = {
        "train_micro_batch_size_per_gpu": 2,
        "optimizer": {"params": {"lr": 0.001}},
    }
    expected_config = copy.deepcopy(deepspeed_config)
    expected_config["train_micro_batch_size_per_gpu"] = 2
    expected_config["optimizer"]["params"]["lr"] = 0.001
    result = det_deepspeed.overwrite_deepspeed_config(base_ds_config, source_ds_config)
    assert result == expected_config

    # Test load base deepspeed config from json file.
    base_ds_config = str(
        pathlib.Path(__file__).resolve().parent.parent.joinpath("fixtures/ds_config.json")
    )
    result = det_deepspeed.overwrite_deepspeed_config(base_ds_config, source_ds_config)
    assert result == expected_config

    # Test fail invalid base_ds_config argument.
    with pytest.raises(TypeError, match="Expected string or dict for base_ds_config argument."):
        _ = det_deepspeed.overwrite_deepspeed_config([1, 2], source_ds_config)

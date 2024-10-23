# type: ignore
import copy
import json
import os
import pathlib
import shutil
from typing import Iterator

import appdirs
import pytest
import torch
from deepspeed.runtime import config_utils

import determined
import determined.pytorch.deepspeed as det_ds
from determined import pytorch  # noqa: I2041
from determined.pytorch.deepspeed import _trainer  # noqa: I2041
from tests.experiment.fixtures import deepspeed_linear_model  # noqa: I2041

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


# Checks shm size and skips certain tests if it can't be determined or isn't enough.
# TODO: Remove these skips after CI is updated (INFENG-659)
def check_shm_size() -> bool:
    return pathlib.Path("/dev/shm").exists() and shutil.disk_usage("/dev/shm")[0] < 10**8


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

        with pytest.raises(AssertionError, match=r"Distributed backend is not initialized. .*"):
            with det_ds.init() as train_context:
                trial = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, updated_hparams)
                trainer = det_ds.Trainer(trial, train_context)
                trainer.fit(max_length=pytorch.Batch(16))

    def test_manual_init_distributed(self, manual_init_distributed: None):
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_manual_init_distributed"] = True

        with det_ds.init() as train_context:
            trial = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, updated_hparams)
            trainer = det_ds.Trainer(trial, train_context)
            trainer.fit(max_length=pytorch.Batch(16))

        assert torch.distributed.is_initialized()

    def test_linear_model(self) -> None:
        with det_ds.init() as train_context:
            trial = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, self.hparams)
            trainer = det_ds.Trainer(trial, train_context)
            trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))

    def test_manual_grad_acc_metrics(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_manual_grad_acc"] = True

        with det_ds.init() as train_context:
            trial = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, updated_hparams)
            trainer = det_ds.Trainer(trial, train_context)
            trainer.fit(max_length=pytorch.Batch(16))

    def test_fail_manual_grad_acc_metrics(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_fail_manual_grad_acc"] = True

        with pytest.raises(AssertionError, match="did not train for gradient accumulation steps"):
            with det_ds.init() as train_context:
                trial = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, updated_hparams)
                trainer = det_ds.Trainer(trial, train_context)
                trainer.fit(max_length=pytorch.Batch(16))

    def test_custom_dataloader(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_manual_dataloader"] = True

        with det_ds.init() as train_context:
            trial = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, updated_hparams)
            trainer = det_ds.Trainer(trial, train_context)
            trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))

    def test_fail_dataset_repro_check(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_fail_dataset_repro_check"] = True

        with pytest.raises(RuntimeError, match=r".* reproducibility .* disable this check .*"):
            with det_ds.init() as train_context:
                trial = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, updated_hparams)
                trainer = det_ds.Trainer(trial, train_context)
                trainer.fit(max_length=pytorch.Batch(16))

    def test_invalid_valid_dataset(self) -> None:
        with pytest.raises(
            determined.errors.InvalidExperimentException,
            match=r".* train micro batches .* should not be less than .*",
        ):
            with det_ds.init() as train_context:
                trial = deepspeed_linear_model.InvalidValidDatasetTrial(train_context, self.hparams)
                trainer = det_ds.Trainer(trial, train_context)
                trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))

    def test_invalid_train_metric(self) -> None:
        with pytest.raises(
            determined.errors.InvalidExperimentException,
            match=r"train_batch() must return a dictionary .*",
        ):
            with det_ds.init() as train_context:
                trial = deepspeed_linear_model.InvalidTrainMetricTrial(train_context, self.hparams)
                trainer = det_ds.Trainer(trial, train_context)
                trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))

    def test_invalid_valid_metric(self) -> None:
        with pytest.raises(
            determined.errors.InvalidExperimentException,
            match=r"evaluate_batch must return a dictionary .*",
        ):
            with det_ds.init() as train_context:
                trial = deepspeed_linear_model.InvalidValidMetricTrial(train_context, self.hparams)
                trainer = det_ds.Trainer(trial, train_context)
                trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))

    def test_differing_valid_metric_keys(self) -> None:
        with pytest.raises(
            ValueError,
            match=r"Validation metric names must match across all batches of data: .*",
        ):
            with det_ds.init() as train_context:
                trial = deepspeed_linear_model.DifferingValidMetricKeyTrial(
                    train_context, self.hparams
                )
                trainer = det_ds.Trainer(trial, train_context)
                trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))

    def test_fail_multiple_set_mpu(self):
        with pytest.raises(
            determined.errors.InvalidExperimentException,
            match=r"Only one MPU can be passed to DeepSpeedTrialContext.",
        ):
            with det_ds.init() as train_context:
                _ = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, self.hparams)
                train_context.set_mpu(det_ds.make_data_parallel_mpu(train_context.distributed))
                train_context.set_mpu(det_ds.make_data_parallel_mpu(train_context.distributed))

    def test_custom_reducer(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["test_custom_reducer"] = True

        with det_ds.init() as train_context:
            trial = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, updated_hparams)
            trainer = det_ds.Trainer(trial, train_context)
            trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))

    def test_linear_non_scalar_metrics(self) -> None:
        updated_hparams = copy.deepcopy(self.hparams)
        updated_hparams["return_non_scalar_metrics"] = True

        with det_ds.init() as train_context:
            trial = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, updated_hparams)
            trainer = det_ds.Trainer(trial, train_context)
            trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))

    def test_linear_pipeline_model(self) -> None:
        with det_ds.init() as train_context:
            trial = deepspeed_linear_model.LinearPipelineEngineTrial(train_context, self.hparams)
            trainer = det_ds.Trainer(trial, train_context)
            trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))

    def test_two_model_engines(self) -> None:
        with det_ds.init() as train_context:
            trial = deepspeed_linear_model.LinearTwoEngineTrial(train_context, self.hparams)
            trainer = det_ds.Trainer(trial, train_context)
            trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))

    def test_checkpointing_and_restoring(self) -> None:
        with det_ds.init() as train_context:
            trial1 = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, self.hparams)
            trainer = det_ds.Trainer(trial1, train_context)
            assert trial1.checkpoint_uuid is None
            trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))
        with det_ds.init() as train_context:
            trial2 = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, self.hparams)
            trainer = det_ds.Trainer(trial2, train_context)
            assert trial1.checkpoint_uuid is not None
            trainer.fit(
                validation_period=pytorch.Batch(16),
                max_length=pytorch.Batch(16),
                latest_checkpoint=os.path.join(
                    appdirs.user_data_dir("determined"), trial1.checkpoint_uuid
                ),
            )

    def test_restore_invalid_checkpoint(self) -> None:
        with det_ds.init() as train_context:
            trial1 = deepspeed_linear_model.LinearDeepSpeedTrial(train_context, self.hparams)
            trainer = det_ds.Trainer(trial1, train_context)
            assert trial1.checkpoint_uuid is None
            trainer.fit(validation_period=pytorch.Batch(16), max_length=pytorch.Batch(16))

        with det_ds.init() as train_context:
            trial2 = deepspeed_linear_model.LinearTwoEngineTrial(train_context, self.hparams)
            trainer = det_ds.Trainer(trial2, train_context)
            assert trial1.checkpoint_uuid is not None
            with pytest.raises(AssertionError, match="Failed to load deepspeed checkpoint."):
                trainer.fit(
                    validation_period=pytorch.Batch(16),
                    max_length=pytorch.Batch(16),
                    latest_checkpoint=os.path.join(
                        appdirs.user_data_dir("determined"), trial1.checkpoint_uuid
                    ),
                )

    @pytest.mark.skipif(check_shm_size(), reason="insufficient shm size")
    def test_reproducibility(self) -> None:
        with det_ds.init() as train_context:
            _trainer._set_random_seeds(self.trial_seed)
            train_context._trial_seed = self.trial_seed
            trial1 = deepspeed_linear_model.LinearPipelineEngineTrial(train_context, self.hparams)
            trainer = det_ds.Trainer(trial1, train_context)
            trainer.fit(validation_period=pytorch.Batch(100), max_length=pytorch.Batch(1000))

        with det_ds.init() as train_context:
            _trainer._set_random_seeds(self.trial_seed)
            train_context._trial_seed = self.trial_seed
            trial2 = deepspeed_linear_model.LinearPipelineEngineTrial(train_context, self.hparams)
            trainer = det_ds.Trainer(trial2, train_context)
            trainer.fit(validation_period=pytorch.Batch(100), max_length=pytorch.Batch(1000))

        assert len(trial1.avg_metrics) == len(trial2.avg_metrics)
        for A, B in zip(trial1.avg_metrics, trial2.avg_metrics):
            assert A.keys() == B.keys()
            for key in A.keys():
                assert abs(A[key] - B[key]) < 10e-7

        assert len(trial1.batch_metrics) == len(trial2.batch_metrics)
        for batch_idx in range(len(trial1.batch_metrics)):
            for A, B in zip(trial1.batch_metrics[batch_idx], trial2.batch_metrics[batch_idx]):
                assert A.keys() == B.keys()
                for key in A.keys():
                    assert abs(A[key] - B[key]) < 10e-7

        assert len(trial1.val_metrics) == len(trial2.val_metrics)
        for A, B in zip(trial1.val_metrics, trial2.val_metrics):
            assert A.keys() == B.keys()
            for key in A.keys():
                assert abs(A[key] - B[key]) < 10e-7

    def test_callbacks(self) -> None:
        with det_ds.init() as train_context:
            trial = deepspeed_linear_model.LinearCallbackTrial(train_context, self.hparams)
            trainer = det_ds.Trainer(trial, train_context)
            trainer.fit(max_length=pytorch.Epoch(2))
            assert trial.counter.__dict__ == {
                "trial_startups": 1,
                "validation_steps_started": 1,
                "validation_steps_ended": 1,
                "checkpoints_written": 1,
                "checkpoints_uploaded": 1,
                "training_started_times": 1,
                "training_epochs_started": 2,
                "training_epochs_ended": 2,
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
    result = det_ds.overwrite_deepspeed_config(base_ds_config, source_ds_config)
    assert result == expected_config

    # Test load base deepspeed config from json file.
    base_ds_config = str(
        pathlib.Path(__file__).resolve().parent.parent.joinpath("fixtures/ds_config.json")
    )
    result = det_ds.overwrite_deepspeed_config(base_ds_config, source_ds_config)
    assert result == expected_config

    # Test fail invalid base_ds_config argument.
    with pytest.raises(TypeError, match="Expected string or dict for base_ds_config argument."):
        _ = det_ds.overwrite_deepspeed_config([1, 2], source_ds_config)

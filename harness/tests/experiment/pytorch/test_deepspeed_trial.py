# type: ignore
import pathlib

from determined import workload
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import deepspeed_linear_model

deepspeed_config = {
    "train_batch_size": 16,
    "train_micro_batch_size_per_gpu": 4,
    "optimizer": {"type": "SGD", "params": {"lr": 0.001, "weight_decay": 3e-7}},
    "scheduler": {
        "type": "WarmupLR",
        "params": {"warmup_min_lr": 0, "warmup_max_lr": 0.001, "warmup_num_steps": 1000},
    },
    "gradient_clipping": 1.0,
    "prescale_gradients": False,
    "fp16": {
        "enabled": False,
    },
    "zero_optimization": {
        "stage": 0,
    },
}


class TestDeepSpeedTrial:
    def setup_method(self) -> None:
        # This training setup is not guaranteed to converge in general,
        # but has been tested with this random seed.  If changing this
        # random seed, verify the initial conditions converge.
        self.trial_seed = 17
        self.hparams = {
            "global_batch_size": 16,
            "deepspeed_config": deepspeed_config,
            "disable_dataset_reproducibility_checks": False,
            "disable_auto_grad_accumulation": False,
            "return_non_scalar_metrics": False,
            "test_custom_reducer": False,
        }

    def test_linear_model(self) -> None:
        def make_workloads() -> workload.Stream:
            trainer = utils.TrainAndValidate()
            train_batch_calls = (
                deepspeed_config["train_batch_size"]
                / deepspeed_config["train_micro_batch_size_per_gpu"]
            )

            yield from trainer.send(
                steps=10, validation_freq=10, train_batch_calls=train_batch_calls
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

            train_batch_calls = (
                deepspeed_config["train_batch_size"]
                / deepspeed_config["train_micro_batch_size_per_gpu"]
            )

            yield from trainer.send(steps=1, validation_freq=1, train_batch_calls=train_batch_calls)
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

    def test_callbacks(self, tmp_path: pathlib.Path) -> None:
        checkpoint_dir = tmp_path.joinpath("checkpoint")
        latest_checkpoint = None
        latest_batch = 0

        controller = None

        def make_workloads1() -> workload.Stream:
            nonlocal controller

            yield workload.train_workload(1, 1, 0, 4), workload.ignore_workload_response
            assert controller is not None, "controller was never set!"
            assert controller.trial.counter.__dict__ == {
                "validation_steps_started": 0,
                "validation_steps_ended": 0,
                "checkpoints_ended": 0,
                "training_started_times": 1,
                "training_epochs_started": 2,
                "training_epochs_ended": 2,
            }

            yield workload.validation_workload(), workload.ignore_workload_response
            assert controller.trial.counter.__dict__ == {
                "validation_steps_started": 1,
                "validation_steps_ended": 1,
                "checkpoints_ended": 0,
                "training_started_times": 1,
                "training_epochs_started": 2,
                "training_epochs_ended": 2,
            }

            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint, latest_batch
            latest_checkpoint = interceptor.metrics_result()["uuid"]
            latest_batch = 1
            assert controller.trial.counter.__dict__ == {
                "validation_steps_started": 1,
                "validation_steps_ended": 1,
                "checkpoints_ended": 1,
                "training_started_times": 1,
                "training_epochs_started": 2,
                "training_epochs_ended": 2,
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

        # Verify the checkpoint loading callback works.

        def make_workloads2() -> workload.Stream:
            yield workload.train_workload(1, 1, 0, 2), workload.ignore_workload_response

        controller = utils.make_trial_controller_from_trial_implementation(
            trial_class=deepspeed_linear_model.LinearCallbackTrial,
            hparams=self.hparams,
            workloads=make_workloads2(),
            checkpoint_dir=str(checkpoint_dir),
            latest_checkpoint=latest_checkpoint,
            latest_batch=latest_batch,
            expose_gpus=True,
        )
        controller.run()
        assert controller.trial.counter.__dict__ == {
            "validation_steps_started": 1,
            "validation_steps_ended": 1,
            "checkpoints_ended": 0,
            "training_started_times": 2,
            "training_epochs_started": 3,
            "training_epochs_ended": 3,
        }

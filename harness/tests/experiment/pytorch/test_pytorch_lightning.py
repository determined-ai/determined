import pathlib
import typing

import pytest
import torch

import determined as det
from determined import pytorch, workload
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import pytorch_xor_model


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


    def test_checkpointing_and_restoring(self, tmp_path: pathlib.Path) -> None:
        def make_trial_controller_fn(
            workloads: workload.Stream, load_path: typing.Optional[str] = None
        ) -> det.TrialController:
            updated_hparams = {
                "lr_scheduler_step_mode": pytorch.LRScheduler.StepMode.STEP_EVERY_BATCH.value,
                **self.hparams,
            }
            return utils.make_trial_controller_from_trial_implementation(
                trial_class=pytorch_xor_model.XORTrialWithLRScheduler,
                hparams=updated_hparams,
                workloads=workloads,
                load_path=load_path,
                trial_seed=self.trial_seed,
            )

        utils.checkpointing_and_restoring_test(make_trial_controller_fn, tmp_path)

    def test_restore_invalid_checkpoint(self, tmp_path: pathlib.Path) -> None:
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


def test_create_trial_instance() -> None:
    utils.create_trial_instance(pytorch_xor_model.XORTrial)

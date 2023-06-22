# type: ignore
import os
import pathlib
import random
import sys
import typing
import uuid

from torch.distributed import launcher

import determined as det
from determined import gpu, pytorch
from tests.experiment import utils


def create_trial_and_trial_controller(
    trial_class: pytorch.PyTorchTrial,
    hparams: typing.Dict,
    slots_per_trial: int = 1,
    scheduling_unit: int = 1,
    trial_seed: int = 17,
    exp_config: typing.Optional[typing.Dict] = None,
    checkpoint_dir: typing.Optional[str] = None,
    tensorboard_path: typing.Optional[pathlib.Path] = None,
    latest_checkpoint: typing.Optional[str] = None,
    steps_completed: int = 0,
    expose_gpus: bool = True,
    max_batches: int = 100,
    min_checkpoint_batches: int = sys.maxsize,
    min_validation_batches: int = sys.maxsize,
    aggregation_frequency: int = 1,
) -> typing.Tuple[pytorch.PyTorchTrial, pytorch._PyTorchTrialController]:
    assert issubclass(
        trial_class, pytorch.PyTorchTrial
    ), "pytorch test method called for non-pytorch trial"

    if not exp_config:
        assert hasattr(
            trial_class, "_searcher_metric"
        ), "Trial classes for unit tests should be annotated with a _searcher_metric attribute"
        searcher_metric = trial_class._searcher_metric
        exp_config = utils.make_default_exp_config(
            hparams, scheduling_unit, searcher_metric, checkpoint_dir=checkpoint_dir
        )

    if not trial_seed:
        trial_seed = random.randint(0, 1 << 31)

    checkpoint_dir = checkpoint_dir or "/tmp"

    distributed_backend = det._DistributedBackend()
    if distributed_backend.use_torch():
        distributed_context = det.core.DistributedContext.from_torch_distributed()
    else:
        distributed_context = None

    core_context = det.core._dummy_init(
        distributed=distributed_context,
        checkpoint_storage=checkpoint_dir,
        tensorboard_path=tensorboard_path,
    )

    # do what core_context.__enter__ does.
    core_context.preempt.start()
    if core_context._tensorboard_manager is not None:
        core_context._tensorboard_manager.start()

    core_context.train._trial_id = "1"
    distributed_backend = det._DistributedBackend()
    if expose_gpus:
        gpu_uuids = gpu.get_gpu_uuids()
    else:
        gpu_uuids = []

    pytorch._PyTorchTrialController.pre_execute_hook(trial_seed, distributed_backend)
    trial_context = pytorch.PyTorchTrialContext(
        core_context=core_context,
        trial_seed=trial_seed,
        hparams=hparams,
        slots_per_trial=slots_per_trial,
        num_gpus=len(gpu_uuids),
        exp_conf=exp_config,
        aggregation_frequency=aggregation_frequency,
        steps_completed=steps_completed,
        managed_training=True,  # this must be True to put model on GPU
        debug_enabled=False,
    )
    trial_context._set_default_gradient_compression(False)
    trial_context._set_default_average_aggregated_gradients(True)
    trial_inst = trial_class(trial_context)

    trial_controller = pytorch._PyTorchTrialController(
        trial_inst=trial_inst,
        context=trial_context,
        max_length=pytorch.Batch(max_batches),
        checkpoint_period=pytorch.Batch(min_checkpoint_batches),
        validation_period=pytorch.Batch(min_validation_batches),
        searcher_metric_name=trial_class._searcher_metric,
        reporting_period=pytorch.Batch(scheduling_unit),
        local_training=True,
        latest_checkpoint=latest_checkpoint,
        steps_completed=steps_completed,
        smaller_is_better=bool(exp_config["searcher"]["smaller_is_better"]),
        test_mode=False,
        checkpoint_policy=exp_config["checkpoint_policy"],
        step_zero_validation=bool(exp_config["perform_initial_validation"]),
        det_profiler=None,
        global_batch_size=None,
    )

    trial_controller._set_data_loaders()
    trial_controller.state = pytorch._TrialState()

    trial_controller.training_iterator = iter(trial_controller.training_loader)
    return trial_inst, trial_controller


def train_for_checkpoint(
    hparams: typing.Dict,
    trial_class: pytorch.PyTorchTrial,
    tmp_path: pathlib.Path,
    exp_config: typing.Dict,
    slots_per_trial: int = 1,
    steps: int = 1,
) -> int:
    checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
    tensorboard_path = tmp_path.joinpath("tensorboard")

    trial, trial_controller = create_trial_and_trial_controller(
        trial_class=trial_class,
        hparams=hparams,
        slots_per_trial=slots_per_trial,
        exp_config=exp_config,
        max_batches=steps,
        min_validation_batches=steps,
        min_checkpoint_batches=steps,
        checkpoint_dir=checkpoint_dir,
        tensorboard_path=tensorboard_path,
        expose_gpus=True,
    )

    trial_controller.run()

    assert len(os.listdir(checkpoint_dir)) == 1, "trial did not create a checkpoint"

    return trial_controller.state.batches_trained


def train_from_checkpoint(
    hparams: typing.Dict,
    trial_class: pytorch.PyTorchTrial,
    tmp_path: pathlib.Path,
    exp_config: typing.Dict,
    slots_per_trial: int = 1,
    steps: typing.Tuple[int, int] = (1, 1),
    batches_trained: int = 0,
) -> None:
    checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
    tensorboard_path = tmp_path.joinpath("tensorboard")

    num_existing_checkpoints = len(os.listdir(checkpoint_dir))

    trial, trial_controller = create_trial_and_trial_controller(
        trial_class=trial_class,
        hparams=hparams,
        slots_per_trial=slots_per_trial,
        exp_config=exp_config,
        max_batches=steps[0] + steps[1],
        min_validation_batches=steps[0],
        min_checkpoint_batches=sys.maxsize,
        checkpoint_dir=checkpoint_dir,
        tensorboard_path=tensorboard_path,
        latest_checkpoint=os.listdir(checkpoint_dir)[0],
        steps_completed=batches_trained,
        expose_gpus=True,
    )
    trial_controller.run()

    assert (
        len(os.listdir(checkpoint_dir)) == num_existing_checkpoints + 1
    ), "trial did not create a checkpoint"


def train_and_checkpoint(
    hparams: typing.Dict,
    trial_class: pytorch.PyTorchTrial,
    tmp_path: pathlib.Path,
    exp_config: typing.Dict,
    steps: typing.Tuple[int, int] = (1, 1),
) -> None:
    # Trial A: train batches and checkpoint
    steps_completed = train_for_checkpoint(
        hparams=hparams,
        trial_class=trial_class,
        tmp_path=tmp_path,
        exp_config=exp_config,
        steps=steps[0],
    )

    # Trial B: restore from checkpoint and train for more batches
    train_from_checkpoint(
        hparams=hparams,
        trial_class=trial_class,
        tmp_path=tmp_path,
        exp_config=exp_config,
        steps=steps,
        batches_trained=steps_completed,
    )


def setup_torch_distributed(local_procs=2, max_retries=0) -> launcher.LaunchConfig:
    # set up distributed backend.
    os.environ[det._DistributedBackend.TORCH] = str(1)

    rdzv_backend = "c10d"
    rdzv_endpoint = "localhost:29400"
    rdzv_id = str(uuid.uuid4())

    launch_config = launcher.LaunchConfig(
        min_nodes=1,
        max_nodes=1,
        nproc_per_node=local_procs,
        run_id=rdzv_id,
        max_restarts=max_retries,
        rdzv_endpoint=rdzv_endpoint,
        rdzv_backend=rdzv_backend,
    )

    return launch_config

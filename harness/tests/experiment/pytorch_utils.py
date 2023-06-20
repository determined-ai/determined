import pathlib
import random
import sys
import typing

import determined as det
from determined import pytorch, gpu

from tests.experiment import utils


def calculate_gradients(
    batch_size: int = 4, epoch_size: int = 64, num_epochs: int = 3
) -> typing.List[float]:
    # independently compute expected metrics
    batches = [
        (v[:], v[:])
        for v in (
            [x * 0.1 + 1.0 for x in range(y, y + batch_size)]
            for y in (z % epoch_size for z in range(0, epoch_size * num_epochs, batch_size))
        )
    ]

    lr = 0.001

    def compute_expected_weight(
        data: typing.List[float], label: typing.List[float], w: float
    ) -> float:
        n = len(data)
        expected_step = 2.0 * lr * sum((d * (l - d * w) for d, l in zip(data, label))) / n
        return w + expected_step

    expected_weights = []
    weight = 0.0
    data: typing.List[float] = []
    label: typing.List[float] = []
    for i, batch in enumerate(batches):
        if i % 2 == 0:
            # for even-numbered batches the optimizer step is a no-op:
            # the weights don't change
            data, label = batch
        else:
            additional_data, additional_label = batch
            data += additional_data
            label += additional_label
            weight = compute_expected_weight(data, label, weight)
        expected_weights.append(weight)

    return expected_weights

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

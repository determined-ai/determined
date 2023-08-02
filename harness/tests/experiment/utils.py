import importlib
import os
import pathlib
import re
import unittest.mock
from typing import Any, Callable, Dict, List, Optional, Sequence, Tuple, Type

import numpy as np
import pytest
from mypy_extensions import DefaultNamedArg

import determined as det
from determined import core, gpu, workload
from determined.common import util


class TrainAndValidate:
    """
    Offer a similar interface as WorkloadResponseInterceptor, except let send() yield a whole
    progression of RUN_STEP and COMPUTE_VALIDATION_METRICS, and let result() return the accumulated
    metrics from each.
    """

    def __init__(self, request_stop_step_id: Optional[int] = None) -> None:
        self._training_metrics = None  # type: Optional[List[Dict[str, Any]]]
        self._avg_training_metrics = None  # type: Optional[List[Dict[str, Any]]]
        self._validation_metrics = None  # type: Optional[List[Dict[str, Any]]]
        self.request_stop_step_id = request_stop_step_id
        self._steps_completed = 0

    def send(
        self,
        steps: int,
        validation_freq: int,
        initial_step_id: int = 1,
        scheduling_unit: int = 1,
        train_batch_calls: int = 1,
    ) -> workload.Stream:
        self._training_metrics = []
        self._avg_training_metrics = []
        self._validation_metrics = []
        self._steps_completed = 0
        interceptor = workload.WorkloadResponseInterceptor()

        for step_id in range(initial_step_id, initial_step_id + steps):
            stop_requested = False
            yield from interceptor.send(
                workload.train_workload(
                    step_id,
                    num_batches=scheduling_unit,
                    total_batches_processed=self._steps_completed,
                ),
            )
            metrics = interceptor.metrics_result()
            batch_metrics = metrics["metrics"]["batch_metrics"]
            assert len(batch_metrics) == scheduling_unit * train_batch_calls
            self._training_metrics.extend(batch_metrics)
            self._avg_training_metrics.append(metrics["metrics"]["avg_metrics"])
            self._steps_completed += scheduling_unit
            if metrics.get("stop_requested"):
                assert step_id == self.request_stop_step_id, (step_id, self)
                stop_requested = True

            if step_id % validation_freq == 0:
                yield from interceptor.send(
                    workload.validation_workload(
                        step_id, total_batches_processed=self._steps_completed
                    ),
                )
                validation = interceptor.metrics_result()
                v_metrics = validation["metrics"]["validation_metrics"]
                self._validation_metrics.append(v_metrics)
                if validation.get("stop_requested"):
                    assert step_id == self.request_stop_step_id, (step_id, self)
                    stop_requested = True

            if stop_requested:
                break
            else:
                assert step_id != self.request_stop_step_id

    def result(self) -> Tuple[List[Dict[str, Any]], List[Dict[str, Any]]]:
        assert self._training_metrics is not None
        assert self._validation_metrics is not None
        return self._training_metrics, self._validation_metrics

    def get_steps_completed(self) -> int:
        return self._steps_completed

    def get_avg_training_metrics(self) -> List[Dict[str, Any]]:
        assert self._avg_training_metrics is not None
        return self._avg_training_metrics


def make_default_exp_config(
    hparams: Dict[str, Any],
    scheduling_unit: int,
    searcher_metric: str,
    checkpoint_dir: Optional[str] = None,
) -> Dict:
    return {
        "scheduling_unit": scheduling_unit,
        "resources": {"native_parallel": False, "slots_per_trial": 1},
        "hyperparameters": hparams,
        "optimizations": {
            "mixed_precision": "O0",
            "aggregation_frequency": 1,
            "gradient_compression": False,
            "average_training_metrics": True,
            "auto_tune_tensor_fusion": False,
            "tensor_fusion_threshold": 100,
            "tensor_fusion_cycle_time": 3.5,
        },
        "data_layer": {"type": "shared_fs"},
        "checkpoint_policy": "best",
        "perform_initial_validation": False,
        "checkpoint_storage": {
            "type": "shared_fs",
            "host_path": checkpoint_dir or "/tmp",
        },
        "searcher": {
            "metric": searcher_metric,
            "smaller_is_better": True,
        },
        "min_checkpoint_period": {"batches": 0},
        "min_validation_period": {"batches": 0},
    }


def make_default_env_context(
    hparams: Dict[str, Any],
    experiment_config: Dict,
    trial_seed: int = 0,
    latest_checkpoint: Optional[str] = None,
    steps_completed: int = 0,
    expose_gpus: bool = False,
) -> det.EnvContext:
    assert (latest_checkpoint is None) == (steps_completed == 0)

    if expose_gpus:
        gpu_uuids = gpu.get_gpu_uuids()
        use_gpu = bool(gpu_uuids)
    else:
        gpu_uuids = []
        use_gpu = False

    return det.EnvContext(
        experiment_config=experiment_config,
        master_url="",
        master_cert_file=None,
        master_cert_name=None,
        hparams=hparams,
        latest_checkpoint=latest_checkpoint,
        steps_completed=steps_completed,
        use_gpu=use_gpu,
        container_gpus=gpu_uuids,
        slot_ids=[],
        debug=False,
        det_trial_id="1",
        det_experiment_id="1",
        det_agent_id="1",
        det_cluster_id="uuid-123",
        trial_seed=trial_seed,
        trial_run_id=1,
        allocation_id="",
        managed_training=True,
        test_mode=False,
        on_cluster=False,
    )


def fixtures_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "fixtures", path)


def repo_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../../", path)


def cv_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../../examples/computer_vision", path)


def gan_examples_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../../examples/gan", path)


def tutorials_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../../examples/tutorials", path)


def features_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../../examples/features", path)


def import_class_from_module(class_name: str, module_path: str) -> Any:
    module_dir = pathlib.Path(os.path.dirname(module_path))

    with det.import_from_path(module_dir):
        spec = importlib.util.spec_from_file_location(class_name, module_path)
        module = importlib.util.module_from_spec(spec)  # type: ignore
        spec.loader.exec_module(module)  # type: ignore
        trial_cls = getattr(module, class_name)  # noqa: B009

    return trial_cls


def load_config(config_path: str) -> Any:
    with open(config_path) as f:
        config = util.safe_load_yaml_with_exceptions(f)
    return config


def assert_equivalent_metrics(metrics_A: Dict[str, Any], metrics_B: Dict[str, Any]) -> None:
    """
    Helper function to verify that two dictionaries of metrics are equivalent
    to each other.
    """
    assert set(metrics_A.keys()) == set(metrics_B.keys())
    for key in metrics_A.keys():
        if isinstance(metrics_A[key], (float, np.float64)):
            assert metrics_A[key] == pytest.approx(metrics_B[key])
        elif isinstance(metrics_A[key], np.ndarray):
            assert np.array_equal(metrics_A[key], metrics_B[key])
        else:
            assert metrics_A[key] == metrics_B[key]


def make_trial_controller_from_trial_implementation(
    trial_class: Type[det.Trial],
    hparams: Dict,
    workloads: Optional[workload.Stream] = None,
    scheduling_unit: int = 1,
    trial_seed: int = 0,
    exp_config: Optional[Dict] = None,
    checkpoint_dir: Optional[str] = None,
    latest_checkpoint: Optional[str] = None,
    steps_completed: int = 0,
    expose_gpus: bool = False,
) -> det.TrialController:
    if not exp_config:
        assert hasattr(
            trial_class, "_searcher_metric"
        ), "Trial classes for unit tests should be annotated with a _searcher_metric attribute"
        searcher_metric = trial_class._searcher_metric  # type: ignore
        exp_config = make_default_exp_config(
            hparams, scheduling_unit, searcher_metric, checkpoint_dir=checkpoint_dir
        )
    env = make_default_env_context(
        hparams=hparams,
        experiment_config=exp_config,
        trial_seed=trial_seed,
        latest_checkpoint=latest_checkpoint,
        steps_completed=steps_completed,
        expose_gpus=expose_gpus,
    )

    checkpoint_dir = checkpoint_dir or "/tmp"
    tbd_path = pathlib.Path(os.path.join("/tmp", "tensorboard"))
    core_context = core._dummy_init(checkpoint_storage=checkpoint_dir, tensorboard_path=tbd_path)

    distributed_backend = det._DistributedBackend()

    controller_class = trial_class.trial_controller_class
    assert controller_class is not None
    controller_class.pre_execute_hook(env, distributed_backend)

    trial_context = trial_class.trial_context_class(core_context, env)
    trial_inst = trial_class(trial_context)

    return controller_class.from_trial(
        trial_inst=trial_inst,
        context=trial_context,
        env=env,
        workloads=workloads,
    )


def reproducibility_test(
    controller_fn: Callable[[workload.Stream], det.TrialController],
    steps: int,
    validation_freq: int,
    seed: int = 123,
    scheduling_unit: int = 1,
) -> Tuple[
    Tuple[Sequence[Dict[str, Any]], Sequence[Dict[str, Any]]],
    Tuple[Sequence[Dict[str, Any]], Sequence[Dict[str, Any]]],
]:
    training_metrics = {}
    validation_metrics = {}

    def make_workloads(tag: str) -> workload.Stream:
        nonlocal training_metrics
        nonlocal validation_metrics

        trainer = TrainAndValidate()

        yield from trainer.send(steps, validation_freq, scheduling_unit=scheduling_unit)
        tm, vm = trainer.result()

        training_metrics[tag] = tm
        validation_metrics[tag] = vm

    # Trial A
    os.environ["DET_TRIAL_SEED"] = str(seed)
    controller_A = controller_fn(make_workloads("A"))
    controller_A.run()

    # Trial B
    assert os.environ["DET_TRIAL_SEED"] == str(seed)
    controller_B = controller_fn(make_workloads("B"))
    controller_B.run()

    assert len(training_metrics["A"]) == len(training_metrics["B"])
    for A, B in zip(training_metrics["A"], training_metrics["B"]):
        assert_equivalent_metrics(A, B)

    assert len(validation_metrics["A"]) == len(validation_metrics["B"])
    for A, B in zip(validation_metrics["A"], validation_metrics["B"]):
        assert_equivalent_metrics(A, B)

    return (
        (training_metrics["A"], validation_metrics["A"]),
        (training_metrics["B"], validation_metrics["B"]),
    )


RestorableMakeControllerFn = Callable[
    [
        workload.Stream,
        DefaultNamedArg(Optional[str], "checkpoint_dir"),  # noqa: F821
        DefaultNamedArg(Optional[str], "latest_checkpoint"),  # noqa: F821
        DefaultNamedArg(int, "steps_completed"),  # noqa: F821
    ],
    det.TrialController,
]


def train_and_validate(
    make_trial_controller_fn: Callable[[workload.Stream], det.TrialController],
    steps: int = 2,
) -> Tuple[Sequence[Dict[str, Any]], Sequence[Dict[str, Any]]]:
    metrics: Dict[str, Any] = {"training": [], "validation": []}

    def make_workloads(steps: int) -> workload.Stream:
        trainer = TrainAndValidate()

        yield from trainer.send(steps, validation_freq=1, scheduling_unit=10)
        tm, vm = trainer.result()
        metrics["training"] += tm
        metrics["validation"] += vm

    controller = make_trial_controller_fn(make_workloads(steps))
    controller.run()

    return (metrics["training"], metrics["validation"])


def checkpointing_and_restoring_test(
    make_trial_controller_fn: RestorableMakeControllerFn,
    tmp_path: pathlib.Path,
    steps: Tuple[int, int] = (1, 1),
    scheduling_unit: int = 100,
) -> Tuple[Sequence[Dict[str, Any]], Sequence[Dict[str, Any]]]:
    """
    Tests if a trial controller of any framework can checkpoint and restore from that checkpoint
    without state changes.

    This test runs two trials.
    1) Trial A runs for one step of 100 batches, checkpoints itself, and restores from
       that checkpoint.
    2) Trial B runs for two steps of 100 batches.

    This test compares the training and validation metrics history of the two trials.
    """

    training_metrics = {"A": [], "B": []}  # type: Dict[str, List[workload.Metrics]]
    validation_metrics = {"A": [], "B": []}  # type: Dict[str, List[workload.Metrics]]
    checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
    latest_checkpoint = None
    steps_completed = 0

    def make_workloads(steps: int, tag: str, checkpoint: bool) -> workload.Stream:
        trainer = TrainAndValidate()

        yield from trainer.send(steps, validation_freq=1, scheduling_unit=scheduling_unit)
        tm, vm = trainer.result()
        training_metrics[tag] += tm
        validation_metrics[tag] += vm

        if checkpoint is not None:
            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint, steps_completed
            latest_checkpoint = interceptor.metrics_result()["uuid"]
            steps_completed = trainer.get_steps_completed()

    controller_A1 = make_trial_controller_fn(
        make_workloads(steps[0], "A", True),
        checkpoint_dir=checkpoint_dir,
    )
    controller_A1.run()
    assert latest_checkpoint is not None, "make_workloads did not set the latest_checkpoint"

    controller_A2 = make_trial_controller_fn(
        make_workloads(steps[1], "A", False),
        checkpoint_dir=checkpoint_dir,
        latest_checkpoint=latest_checkpoint,
        steps_completed=steps_completed,
    )
    controller_A2.run()

    controller_B = make_trial_controller_fn(make_workloads(steps[0] + steps[1], "B", False))
    controller_B.run()

    for A, B in zip(training_metrics["A"], training_metrics["B"]):
        assert_equivalent_metrics(A, B)

    for A, B in zip(validation_metrics["A"], validation_metrics["B"]):
        assert_equivalent_metrics(A, B)

    return (training_metrics["A"], training_metrics["B"])


def list_all_files(directory: str) -> List[str]:
    return [f for _, _, files in os.walk(directory) for f in files]


def ensure_requires_global_batch_size(
    trial_class: Type[det.Trial],
    hparams: Dict[str, Any],
) -> None:
    bad_hparams = dict(hparams)
    del bad_hparams["global_batch_size"]

    def make_workloads() -> workload.Stream:
        trainer = TrainAndValidate()
        yield from trainer.send(steps=1, validation_freq=1)

    # Catch missing global_batch_size.
    with pytest.raises(det.errors.InvalidExperimentException, match="is a required hyperparameter"):
        _ = make_trial_controller_from_trial_implementation(
            trial_class, workloads=make_workloads(), hparams=bad_hparams
        )


def assert_patterns_in_logs(input_list: List[str], patterns: List[str]) -> None:
    """
    Match each regex pattern in the list to the logs, one-at-a-time, in order.
    """
    assert patterns, "must provide at least one pattern"
    patterns_iter = iter(patterns)
    p = re.compile(next(patterns_iter))
    for log_line in input_list:
        if p.search(log_line) is None:
            continue
        # Matched a pattern.
        try:
            p = re.compile(next(patterns_iter))
        except StopIteration:
            # All patterns have been matched.
            return
    # Some patterns were not found.
    text = '"\n  "'.join([p.pattern, *patterns_iter])
    raise ValueError(
        f'the following patterns:\n  "{text}"\nwere not found in \
        the trial logs:\n\n{"".join(input_list)}'  # noqa
    )


def get_mock_distributed_context(
    rank: int = 0,
    all_gather_return_value: Optional[Any] = None,
    gather_return_value: Optional[Any] = None,
) -> unittest.mock.MagicMock:
    mock_distributed_context = unittest.mock.MagicMock()
    mock_distributed_context.get_rank.return_value = rank
    mock_distributed_context.broadcast.return_value = "mock_checkpoint_uuid"
    mock_distributed_context.allgather.return_value = all_gather_return_value
    mock_distributed_context.gather.return_value = gather_return_value
    return mock_distributed_context

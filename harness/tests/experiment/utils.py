import distutils.util
import os
import pathlib
import tempfile
from typing import Any, Callable, Dict, List, Optional, Sequence, Tuple, Type

import numpy as np
import pytest
from mypy_extensions import DefaultNamedArg
from tensorflow.keras import utils as keras_utils

import determined as det
from determined import constants, experimental, gpu, horovod, keras, load, workload
from determined.common import check
from determined.common.types import ExperimentID, StepID, TrialID


class TrainAndValidate:
    """
    Offer a similar interface as WorkloadResponseInterceptor, execpt let send() yield a whole
    progression of RUN_STEP and COMPUTE_VALIDATION_METRICS, and let result() return the accumulated
    metrics from each.
    """

    def __init__(self, request_stop_step_id: Optional[int] = None) -> None:
        self._training_metrics = None  # type: Optional[List[Dict[str, Any]]]
        self._avg_training_metrics = None  # type: Optional[List[Dict[str, Any]]]
        self._validation_metrics = None  # type: Optional[List[Dict[str, Any]]]
        self.request_stop_step_id = request_stop_step_id

    def send(
        self, steps: int, validation_freq: int, initial_step_id: int = 1, scheduling_unit: int = 1
    ) -> workload.Stream:
        self._training_metrics = []
        self._avg_training_metrics = []
        self._validation_metrics = []
        total_batches_processed = 0
        interceptor = workload.WorkloadResponseInterceptor()

        for step_id in range(initial_step_id, initial_step_id + steps):
            stop_requested = False
            yield from interceptor.send(
                workload.train_workload(
                    step_id,
                    num_batches=scheduling_unit,
                    total_batches_processed=total_batches_processed,
                ),
            )
            metrics = interceptor.metrics_result()
            batch_metrics = metrics["metrics"]["batch_metrics"]
            assert len(batch_metrics) == scheduling_unit
            self._training_metrics.extend(batch_metrics)
            self._avg_training_metrics.append(metrics["metrics"]["avg_metrics"])
            total_batches_processed += scheduling_unit
            if metrics.get("exited_reason") == "USER_CANCELED":
                assert step_id == self.request_stop_step_id
                stop_requested = True

            if step_id % validation_freq == 0:
                yield from interceptor.send(
                    workload.validation_workload(
                        step_id, total_batches_processed=total_batches_processed
                    ),
                )
                validation = interceptor.metrics_result()
                v_metrics = validation["metrics"]["validation_metrics"]
                self._validation_metrics.append(v_metrics)
                if validation.get("exited_reason") == "USER_CANCELED":
                    assert step_id == self.request_stop_step_id
                    stop_requested = True

            if stop_requested:
                break
            else:
                assert step_id != self.request_stop_step_id

    def result(self) -> Tuple[List[Dict[str, Any]], List[Dict[str, Any]]]:
        assert self._training_metrics is not None
        assert self._validation_metrics is not None
        return self._training_metrics, self._validation_metrics

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
            "average_training_metrics": False,
        },
        "data_layer": {"type": "shared_fs"},
        "checkpoint_storage": {
            "type": "shared_fs",
            "host_path": checkpoint_dir or "/tmp",
        },
        "searcher": {
            "metric": searcher_metric,
        },
    }


def make_default_env_context(
    hparams: Dict[str, Any],
    experiment_config: Dict,
    trial_seed: int = 0,
    latest_checkpoint: Optional[Dict[str, Any]] = None,
) -> det.EnvContext:
    # TODO(ryan): Fix the parameter passing so that this doesn't read from environment variables,
    # and we can get rid of the @expose_gpus fixture.
    use_gpu = distutils.util.strtobool(os.environ.get("DET_USE_GPU", "false"))
    gpu_uuids = gpu.get_gpu_uuids_and_validate(use_gpu)

    return det.EnvContext(
        experiment_config=experiment_config,
        initial_workload=workload.Workload(
            workload.Workload.Kind.RUN_STEP,
            ExperimentID(1),
            TrialID(1),
            StepID(1),
            det.ExperimentConfig(experiment_config).scheduling_unit(),
            0,
        ),
        master_addr="",
        master_port=0,
        use_tls=False,
        master_cert_file=None,
        master_cert_name=None,
        container_id="",
        hparams=hparams,
        latest_checkpoint=latest_checkpoint,
        use_gpu=use_gpu,
        container_gpus=gpu_uuids,
        slot_ids=[],
        debug=False,
        workload_manager_type="TRIAL_WORKLOAD_MANAGER",
        det_rendezvous_port="",
        det_trial_unique_port_offset=0,
        det_trial_runner_network_interface=constants.AUTO_DETECT_TRIAL_RUNNER_NETWORK_INTERFACE,
        det_trial_id="1",
        det_experiment_id="1",
        det_agent_id="1",
        det_cluster_id="uuid-123",
        det_task_token="",
        trial_seed=trial_seed,
        managed_training=True,
        test_mode=False,
        on_cluster=False,
    )


def make_default_rendezvous_info() -> det.RendezvousInfo:
    return det.RendezvousInfo(addrs=[f"127.0.0.1:{constants.LOCAL_RENDEZVOUS_PORT}"], rank=0)


def make_default_hvd_config() -> horovod.HorovodContext:
    return horovod.HorovodContext(
        use=False,
        aggregation_frequency=1,
        fp16_compression=False,
        grad_updates_size_file="",
        average_aggregated_gradients=True,
        average_training_metrics=False,
    )


def fixtures_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "fixtures", path)


def repo_path(path: str) -> str:
    return os.path.join(os.path.dirname(__file__), "../../../", path)


def assert_equivalent_metrics(metrics_A: Dict[str, Any], metrics_B: Dict[str, Any]) -> None:
    """
    Helper function to verify that two dictionaries of metrics are equivalent
    to each other.
    """
    assert set(metrics_A.keys()) == set(metrics_B.keys())
    for key in metrics_A.keys():
        if isinstance(metrics_A[key], (float, np.float)):
            assert metrics_A[key] == pytest.approx(metrics_B[key])
        elif isinstance(metrics_A[key], np.ndarray):
            assert np.array_equal(metrics_A[key], metrics_B[key])
        else:
            assert metrics_A[key] == metrics_B[key]


def xor_data(dtype: np.dtype = np.int64) -> Tuple[np.ndarray, np.ndarray]:
    training_data = np.array([[0, 0], [0, 1], [1, 0], [1, 1]], dtype=dtype)
    training_labels = np.array([0, 1, 1, 0], dtype=dtype)
    return training_data, training_labels


def make_xor_data_sequences(
    shuffle: bool = False,
    seed: Optional[int] = None,
    dtype: np.dtype = np.int64,
    multi_input_output: bool = False,
    batch_size: int = 1,
) -> Tuple[keras_utils.Sequence, keras_utils.Sequence]:
    """
    Generates data loaders for the toy XOR problem.  The dataset only has four
    possible inputs.  For the purposes of testing, the validation set is the
    same as the training dataset.
    """
    training_data, training_labels = xor_data(dtype)

    if shuffle:
        if seed is not None:
            np.random.seed(seed)
        idxs = np.random.permutation(4)
        training_data = training_data[idxs]
        training_labels = training_labels[idxs]

    return (
        keras._ArrayLikeAdapter(training_data, training_labels, batch_size=batch_size),
        keras._ArrayLikeAdapter(training_data, training_labels, batch_size=batch_size),
    )


def make_trial_controller_from_trial_implementation(
    trial_class: Type[det.Trial],
    hparams: Dict,
    workloads: workload.Stream,
    scheduling_unit: int = 1,
    trial_seed: int = 0,
    exp_config: Optional[Dict] = None,
    checkpoint_dir: Optional[str] = None,
    latest_checkpoint: Optional[Dict[str, Any]] = None,
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
    )

    rendezvous_info = make_default_rendezvous_info()
    hvd_config = make_default_hvd_config()

    return load.load_trial(
        trial_class=trial_class,
        env=env,
        workloads=workloads,
        rendezvous_info=rendezvous_info,
        hvd_config=hvd_config,
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

        yield workload.terminate_workload(), workload.ignore_workload_response

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
        DefaultNamedArg(Optional[Dict], "latest_checkpoint"),  # noqa: F821
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

        yield workload.terminate_workload(), workload.ignore_workload_response

    controller = make_trial_controller_fn(make_workloads(steps))
    controller.run()

    return (metrics["training"], metrics["validation"])


def checkpointing_and_restoring_test(
    make_trial_controller_fn: RestorableMakeControllerFn, tmp_path: pathlib.Path
) -> Tuple[Sequence[Dict[str, Any]], Sequence[Dict[str, Any]]]:
    """
    Tests if a trial controller of any framework can checkpoint and restore from that checkpoint
    without state changes.

    This test runs two trials.
    1) Trial A runs for one steps of 100 batches, checkpoints itself, and restores from
       that checkpoint.
    2) Trial B runs for two steps of 100 batches.

    This test compares the training and validation metrics history of the two trials.
    """

    training_metrics = {"A": [], "B": []}  # type: Dict[str, List[workload.Metrics]]
    validation_metrics = {"A": [], "B": []}  # type: Dict[str, List[workload.Metrics]]
    checkpoint_dir = str(tmp_path.joinpath("checkpoint"))
    latest_checkpoint = None

    def make_workloads(steps: int, tag: str, checkpoint: bool) -> workload.Stream:
        trainer = TrainAndValidate()

        yield from trainer.send(steps, validation_freq=1, scheduling_unit=100)
        tm, vm = trainer.result()
        training_metrics[tag] += tm
        validation_metrics[tag] += vm

        if checkpoint is not None:
            interceptor = workload.WorkloadResponseInterceptor()
            yield from interceptor.send(workload.checkpoint_workload())
            nonlocal latest_checkpoint
            latest_checkpoint = interceptor.metrics_result()["metrics"].__json__()

        yield workload.terminate_workload(), workload.ignore_workload_response

    controller_A1 = make_trial_controller_fn(
        make_workloads(1, "A", True),
        checkpoint_dir=checkpoint_dir,
    )
    controller_A1.run()
    assert latest_checkpoint is not None, "make_workloads did not set the latest_checkpoint"

    controller_A2 = make_trial_controller_fn(
        make_workloads(1, "A", False),
        checkpoint_dir=checkpoint_dir,
        latest_checkpoint=latest_checkpoint,
    )
    controller_A2.run()

    controller_B = make_trial_controller_fn(make_workloads(2, "B", False))
    controller_B.run()

    for A, B in zip(training_metrics["A"], training_metrics["B"]):
        assert_equivalent_metrics(A, B)

    for A, B in zip(validation_metrics["A"], validation_metrics["B"]):
        assert_equivalent_metrics(A, B)

    return (training_metrics["A"], training_metrics["B"])


def list_all_files(directory: str) -> List[str]:
    return [f for _, _, files in os.walk(directory) for f in files]


def create_trial_instance(trial_def: Type[det.Trial]) -> None:
    with tempfile.TemporaryDirectory() as td:
        trial_instance = experimental.create_trial_instance(
            trial_def=trial_def,
            config={
                "hyperparameters": {
                    "global_batch_size": det.Constant(16),
                    "hidden_size": 4,
                    "learning_rate": 0.01,
                }
            },
            checkpoint_dir=td,
        )
    check.check_isinstance(trial_instance, det.Trial)

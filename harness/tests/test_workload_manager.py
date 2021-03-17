import contextlib
import os
import pathlib
from typing import Any, Dict, Iterator, Optional, cast

import numpy as np
import pytest

import determined as det
from determined import layers, tensorboard, workload
from determined_common import check, storage
from tests.experiment import utils


class NoopTensorboardManager(tensorboard.TensorboardManager):
    def __init__(self) -> None:
        pass

    def sync(self) -> None:
        pass

    def delete(self) -> None:
        pass


class NoopBatchMetricWriter(tensorboard.BatchMetricWriter):
    def __init__(self) -> None:
        pass

    def on_train_step_end(self, *_: Any, **__: Any) -> None:
        pass

    def on_validation_step_end(self, *_: Any, **__: Any) -> None:
        pass


class NoopStorageManager(storage.StorageManager):
    @contextlib.contextmanager
    def restore_path(self, metadata: storage.StorageMetadata) -> Iterator[str]:
        raise NotImplementedError()


class FailOnUploadStorageManager(storage.StorageManager):
    def post_store_path(
        self, storage_id: str, storage_dir: str, metadata: storage.StorageMetadata
    ) -> None:
        raise ValueError("upload error")

    @contextlib.contextmanager
    def restore_path(self, metadata: storage.StorageMetadata) -> Iterator[str]:
        raise NotImplementedError()


class NoopTrialController(det.TrialController):
    def __init__(
        self, workloads: workload.Stream, validation_metrics: Optional[Dict[str, Any]] = None
    ) -> None:
        self.workloads = workloads
        self.validation_metrics = validation_metrics

    @staticmethod
    def pre_execute_hook(*_: Any, **__: Any) -> Any:
        raise NotImplementedError()

    @staticmethod
    def from_trial(*_: Any, **__: Any) -> det.TrialController:
        raise NotImplementedError()

    @staticmethod
    def from_native(*_: Any, **__: Any) -> det.TrialController:
        raise NotImplementedError()

    def run(self) -> None:
        for w, args, response_func in self.workloads:
            if w.kind == workload.Workload.Kind.RUN_STEP:
                metrics = det.util.make_metrics(
                    num_inputs=None,
                    batch_metrics=[{"loss": 1} for _ in range(w.num_batches)],
                )
                response_func({"metrics": metrics})
            elif w.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                check.len_eq(args, 0)
                response_func({"metrics": {"validation_metrics": self.validation_metrics}})
            elif w.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                check.len_eq(args, 1)
                check.is_instance(args[0], pathlib.Path)
                path = cast(pathlib.Path, args[0])
                if not path.exists():
                    path.mkdir(parents=True, exist_ok=True)
                with path.joinpath("a_file").open("w") as f:
                    f.write("yup")
                response_func({})
            elif w.kind == workload.Workload.Kind.TERMINATE:
                raise NotImplementedError()


def test_checkpoint_upload_failure(tmp_path: pathlib.Path) -> None:
    hparams = {"global_batch_size": 64}
    env = utils.make_default_env_context(hparams)
    rendezvous_info = utils.make_default_rendezvous_info()
    storage_manager = FailOnUploadStorageManager(str(tmp_path))
    tensorboard_manager = NoopTensorboardManager()
    metric_writer = NoopBatchMetricWriter()

    def checkpoint_response_func(metrics: workload.Response) -> None:
        raise ValueError("response_func should not be called if the upload fails")

    def make_workloads() -> workload.Stream:
        yield workload.train_workload(1, num_batches=100), [], workload.ignore_workload_response
        yield workload.checkpoint_workload(), [], checkpoint_response_func

    workload_manager = layers.build_workload_manager(
        env,
        make_workloads(),
        rendezvous_info,
        storage_manager,
        tensorboard_manager,
        metric_writer,
    )

    trial_controller = NoopTrialController(iter(workload_manager))

    # Iterate through the events in the workload_manager as the TrialController would.
    with pytest.raises(ValueError, match="upload error"):
        trial_controller.run()


def test_reject_nonscalar_searcher_metric() -> None:
    metric_name = "validation_error"

    hparams = {"global_batch_size": 64}
    experiment_config = utils.make_default_exp_config(hparams, 1)
    experiment_config["searcher"] = {"metric": metric_name}
    env = utils.make_default_env_context(hparams=hparams, experiment_config=experiment_config)
    rendezvous_info = utils.make_default_rendezvous_info()
    storage_manager = NoopStorageManager(os.devnull)
    tensorboard_manager = NoopTensorboardManager()
    metric_writer = NoopBatchMetricWriter()

    def make_workloads() -> workload.Stream:
        yield workload.train_workload(1, num_batches=100), [], workload.ignore_workload_response
        yield workload.validation_workload(), [], workload.ignore_workload_response

    # Normal Python numbers and NumPy scalars are acceptable; other values are not.
    cases = [
        (True, 17),
        (True, 0.17),
        (True, np.float64(0.17)),
        (True, np.float32(0.17)),
        (False, "foo"),
        (False, [0.17]),
        (False, {}),
    ]
    for is_valid, metric_value in cases:
        workload_manager = layers.build_workload_manager(
            env,
            make_workloads(),
            rendezvous_info,
            storage_manager,
            tensorboard_manager,
            metric_writer,
        )

        trial_controller = NoopTrialController(
            iter(workload_manager), validation_metrics={metric_name: metric_value}
        )
        if is_valid:
            trial_controller.run()
        else:
            with pytest.raises(AssertionError, match="non-scalar"):
                trial_controller.run()

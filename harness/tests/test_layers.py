import contextlib
import pathlib
from typing import Any, Dict, Iterator, Optional, cast

import numpy as np
import pytest

import determined as det
from determined import layers, workload
from determined_common import check, storage
from tests.experiment import utils


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
        for wkld, args, response_func in self.workloads:
            if wkld.kind == workload.Workload.Kind.RUN_STEP:
                metrics = det.util.make_metrics(
                    num_inputs=None,
                    batch_metrics=[{"loss": 1} for _ in range(wkld.num_batches)],
                )
                response_func({"metrics": metrics})
            elif wkld.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                check.len_eq(args, 0)
                response_func({"metrics": {"validation_metrics": self.validation_metrics}})
            elif wkld.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                check.len_eq(args, 1)
                check.is_instance(args[0], pathlib.Path)
                path = cast(pathlib.Path, args[0])
                if not path.exists():
                    path.mkdir(parents=True, exist_ok=True)
                with path.joinpath("a_file").open("w") as f:
                    f.write("yup")
                response_func({})
            elif wkld.kind == workload.Workload.Kind.TERMINATE:
                raise NotImplementedError()


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


class TestStorageManager:
    def make_uploadable_file(self, path: pathlib.Path) -> None:
        # Write a file to make the storage manager happy.
        if not path.exists():
            path.mkdir(parents=True, exist_ok=True)
        with path.joinpath("a_file").open("w") as f:
            f.write("yup")

    def test_set_checkpoint_path(self, tmp_path: pathlib.Path) -> None:
        def make_workloads() -> workload.Stream:
            yield workload.train_workload(1, num_batches=1), [], workload.ignore_workload_response
            yield workload.checkpoint_workload(), [], workload.ignore_workload_response

        storage_manager = NoopStorageManager(str(tmp_path))
        storage_layer = layers.StorageLayer(make_workloads(), storage_manager, is_chief=True)

        for wkld, args, response_func in iter(storage_layer):
            if wkld.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                assert args, "StorageLayer did not set args for the CHECKPOINT_MODEL message."
                # Write a file to make the storage manager happy.
                self.make_uploadable_file(cast(pathlib.Path, args[0]))
            else:
                assert not args, f"StorageLayer did set args for the {wkld.kind} message."
            response_func({})

    def test_checkpoint_upload_failure(self, tmp_path: pathlib.Path) -> None:
        def checkpoint_response_func(metrics: workload.Response) -> None:
            raise ValueError("response_func should not be called if the upload fails")

        def make_workloads() -> workload.Stream:
            yield workload.checkpoint_workload(), [], checkpoint_response_func

        storage_manager = FailOnUploadStorageManager(str(tmp_path))
        storage_layer = layers.StorageLayer(make_workloads(), storage_manager, is_chief=True)

        with pytest.raises(ValueError, match="upload error"):
            for wkld, args, response_func in iter(storage_layer):
                if wkld.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                    # Write a file to make the storage manager happy.
                    self.make_uploadable_file(cast(pathlib.Path, args[0]))
                response_func({})


class TestWorkloadManager:
    def test_reject_nonscalar_searcher_metric(self) -> None:
        metric_name = "validation_error"

        hparams = {"global_batch_size": 64}
        experiment_config = utils.make_default_exp_config(hparams, 1)
        experiment_config["searcher"] = {"metric": metric_name}
        env = utils.make_default_env_context(hparams=hparams, experiment_config=experiment_config)

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
                is_chief=True,
            )

            trial_controller = NoopTrialController(
                iter(workload_manager), validation_metrics={metric_name: metric_value}
            )
            if is_valid:
                trial_controller.run()
            else:
                with pytest.raises(AssertionError, match="non-scalar"):
                    trial_controller.run()

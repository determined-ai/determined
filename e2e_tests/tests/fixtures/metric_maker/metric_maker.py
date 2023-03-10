import math
import pathlib
import pickle
from typing import Any, Dict, Union

import numpy as np

import determined as det
from determined import layers, tensorboard, util, workload


def structure_to_metrics(value: float, structure: Any) -> Any:
    """
    Given a base value and a nested structure, return a matching structure where
    all leaves of the structure have been multiplied by the value.

    The structure can be a number, a numpy array, a list, a dictionary, or a tuple.
    """
    if isinstance(structure, list):
        return [structure_to_metrics(value, s) for s in structure]
    if isinstance(structure, tuple):
        return tuple(structure_to_metrics(value, s) for s in structure)
    if isinstance(structure, dict):
        return {k: structure_to_metrics(value, s) for k, s in structure.items()}

    return structure * value


def structure_equal(a: Any, b: Any) -> bool:
    """
    Confirm two structures are equal. Does not handle floating point error.
    """
    if isinstance(a, np.ndarray) and isinstance(b, np.ndarray):
        if not np.all(a == b):
            print(f"ndarrays not equal: {a} vs {b}")
            return False
        return True

    if isinstance(a, (list, tuple)) and isinstance(b, (list, tuple)):
        if len(a) != len(b):
            return False
        for a_elem, b_elem in zip(a, b):
            if not structure_equal(a_elem, b_elem):
                print(f"lists not equal: {a_elem} vs {b_elem}")
                return False
        return True

    if isinstance(a, dict) and isinstance(b, dict):
        assert set(a.keys()) == set(b.keys())
        for key in a:
            if not structure_equal(a[key], b[key]):
                print(f"dict values for key {key} not equal: {a[key]} vs {b[key]}")
                return False
        return True

    return a == b


class MetricMakerTrialContext(det.TrialContext):
    """
    MetricMakerTrial needs batch sizes.
    """

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._per_slot_batch_size, self._global_batch_size = util.calculate_batch_sizes(
            self.get_hparams(),
            self.env.experiment_config.slots_per_trial(),
            "MetricMakerTrial",
        )

    def get_per_slot_batch_size(self) -> int:
        return self._per_slot_batch_size

    def get_global_batch_size(self) -> int:
        return self._global_batch_size


class DummyMetricWriter(tensorboard.MetricWriter):
    def add_scalar(self, name: str, value: Union[int, float, "np.number"], step: int) -> None:
        pass

    def reset(self) -> None:
        pass


class MetricMaker(det.TrialController):
    """
    MetricMaker is a class designed to test that metrics reported from a trial
    are faithfully passed to the master and stored in the database.

    SearchMethods are already tested with unit tests in the master, which
    ensures that they work properly, given the correct metrics from the trial.
    MetricMaker helps test that the correct metrics are actually passed to the
    trial.

    MetricMaker has support for generating arbitrary structures in the metrics
    based on hyperparameters.
    """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)

        self.value = self.env.hparams["starting_base_value"]
        self.training_structure = self.env.hparams["training_structure"]
        self.validation_structure = self.env.hparams["validation_structure"]
        self.gain_per_batch = self.env.hparams["gain_per_batch"]

        self.wlsq = None
        if self.workloads is None:
            self.workloads, self.wlsq = layers.make_compatibility_workloads(
                self.context._core, self.env, self.context.get_global_batch_size()
            )

        self.steps_completed = self.env.steps_completed

        if self.env.latest_checkpoint is not None:
            with self.context._core.checkpoint.restore_path(
                self.env.latest_checkpoint
            ) as load_path:
                self.load(pathlib.Path(load_path))

    @staticmethod
    def from_trial(trial_inst: det.Trial, *args: Any, **kwargs: Any) -> det.TrialController:
        return MetricMaker(*args, **kwargs)

    @staticmethod
    def pre_execute_hook(env: det.EnvContext, distributed_backend: det._DistributedBackend) -> None:
        pass

    def create_metric_writer(self) -> tensorboard.BatchMetricWriter:
        return tensorboard.BatchMetricWriter(DummyMetricWriter())

    def run(self) -> None:
        for w, response_func in self.workloads:
            if w.kind == workload.Workload.Kind.RUN_STEP:
                response = self.train_for_step(w.step_id, w.num_batches)
            elif w.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                response = self.compute_validation_metrics(w.step_id)
            elif w.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                metadata = {"steps_completed": self.steps_completed}
                if self.is_chief:
                    with self.context._core.checkpoint.store_path(metadata) as (
                        path,
                        storage_id,
                    ):
                        self.save(path)
                        response = {"uuid": storage_id}
                else:
                    response = {}
            else:
                raise AssertionError("Unexpected workload: {}".format(w.kind))

            response_func(response)

    def train_for_step(self, step_id: int, num_batches: int) -> Dict[str, Any]:
        # Get the base value for each batch
        batch_values = self.value + self.gain_per_batch * np.arange(num_batches)

        # Get a training metric structure for each batch.
        batch_metrics = [structure_to_metrics(v, self.training_structure) for v in batch_values]

        # Update the overall base value for the trial.
        self.value += self.gain_per_batch * num_batches

        self.steps_completed += num_batches

        return {
            "metrics": det.util.make_metrics(num_batches, batch_metrics),
            "stop_requested": self.context.get_stop_requested(),
        }

    def compute_validation_metrics(self, step_id: int) -> Dict[str, Any]:
        return {
            "metrics": {
                "validation_metrics": structure_to_metrics(self.value, self.validation_structure)
            },
            "stop_requested": self.context.get_stop_requested(),
        }

    def set_random_seed(self, trial_seed) -> None:
        pass

    def save(self, path: pathlib.Path) -> None:
        """
        Save the current value to a file. This would enable testing of PBT
        metrics, where the overall state of a trial is like a piece-wise
        function which takes into account multiple generations of hparams.
        """
        with path.joinpath("checkpoint_file").open("w") as f:
            f.write(str(self.value))

        wlsq_path = path.joinpath("workload_sequencer.pkl")
        if self.wlsq is not None:
            with wlsq_path.open("wb") as f:
                pickle.dump(self.wlsq.get_state(), f)

    def load(self, path: pathlib.Path) -> None:
        with path.joinpath("checkpoint_file").open("r") as f:
            self.value = float(f.read())

        wlsq_path = path.joinpath("workload_sequencer.pkl")
        if self.wlsq is not None and wlsq_path.exists():
            with wlsq_path.open("rb") as f:
                self.wlsq.load_state(pickle.load(f))


class MetricMakerTrial(det.Trial):
    trial_controller_class = MetricMaker
    trial_context_class = MetricMakerTrialContext

    def __init__(self, context: det.TrialContext) -> None:
        self.context = context


class NANMetricMaker(MetricMaker):
    """
    Insert Infinity and NaN values into metrics
    because YAML->JSON parser cannot convert YAML's .inf value
    """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)

        self.value = self.env.hparams["starting_base_value"]
        self.training_structure = self.env.hparams["training_structure"]
        self.training_structure["inf"] = math.inf
        self.training_structure["nan"] = math.nan
        self.training_structure["nanarray"] = np.array([math.nan, math.nan])
        self.validation_structure = self.env.hparams["validation_structure"]
        self.validation_structure["neg_inf"] = -1 * math.inf
        self.gain_per_batch = 0

        self.wlsq = None
        if self.workloads is None:
            self.workloads, self.wlsq = layers.make_compatibility_workloads(
                self.context._core, self.env, self.context.get_global_batch_size()
            )

    @staticmethod
    def from_trial(trial_inst: det.Trial, *args: Any, **kwargs: Any) -> det.TrialController:
        return NANMetricMaker(*args, **kwargs)


class NANMetricMakerTrial(det.Trial):
    trial_controller_class = NANMetricMaker
    trial_context_class = MetricMakerTrialContext

    def __init__(self, context: det.TrialContext) -> None:
        self.context = context

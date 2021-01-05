import pathlib
from typing import Any, Dict

import numpy as np

import determined as det
from determined import horovod


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


class MetricMaker(det.CallbackTrialController):
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

    @staticmethod
    def from_trial(trial_inst: det.Trial, *args: Any, **kwargs: Any) -> det.TrialController:
        return MetricMaker(*args, **kwargs)

    @staticmethod
    def pre_execute_hook(env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        pass

    def train_for_step(self, step_id: int, num_batches: int) -> Dict[str, Any]:
        # Get the base value for each batch
        batch_values = self.value + self.gain_per_batch * np.arange(num_batches)

        # Get a training metric structure for each batch.
        batch_metrics = [structure_to_metrics(v, self.training_structure) for v in batch_values]

        # Update the overall base value for the trial..
        self.value += self.gain_per_batch * num_batches

        return {"metrics": {"batch_metrics": batch_metrics, "num_inputs": num_batches}}

    def compute_validation_metrics(self, step_id: int) -> Dict[str, Any]:
        return {
            "metrics": {
                "validation_metrics": structure_to_metrics(self.value, self.validation_structure)
            }
        }

    def set_random_seed(self, trial_seed) -> None:
        pass

    def save(self, path: pathlib.Path) -> None:
        """
        Save the current value to a file. This would enable testing of PBT
        metrics, where the overall state of a trial is like a piece-wise
        function which takes into account multiple generations of hparams.
        """
        path.mkdir()
        with path.joinpath("checkpoint_file").open("w") as f:
            f.write(str(self.value))

    def load(self, path: pathlib.Path) -> None:
        with path.joinpath("checkpoint_file").open("r") as f:
            self.value = float(f.read())


class MetricMakerTrial(det.Trial):
    trial_controller_class = MetricMaker

    def __init__(self, context: det.TrialContext) -> None:
        self.context = context

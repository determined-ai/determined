import numpy as np

import model_hub.huggingface as hf
from model_hub import utils


def test_expand_like() -> None:
    array_list = [np.array([[1, 2], [3, 4]]), np.array([[2, 3, 4], [3, 4, 5]])]
    result = utils.expand_like(array_list)
    assert np.array_equal(result, np.array([[1, 2, -100], [3, 4, -100], [2, 3, 4], [3, 4, 5]]))


def test_reducer() -> None:
    def mean_fn(x, y):  # type: ignore
        return np.mean(x), np.mean(y)

    reducer = utils.PredLabelFnReducer(mean_fn)
    reducer.update([[1, 2], [3, 4]], [2, 4])
    reducer.update([[5, 6], [7, 8]], [3, 5])
    result = reducer.cross_slot_reduce([reducer.per_slot_reduce()])
    assert result == (4.5, 3.5)


def test_compute_num_training_steps() -> None:
    experiment_config = {"searcher": {"max_length": {"epochs": 3}}, "records_per_epoch": 124}
    num_training_steps = hf.compute_num_training_steps(experiment_config, 16)
    assert num_training_steps == 21

    experiment_config = {
        "searcher": {"max_length": {"batches": 300}},
    }
    num_training_steps = hf.compute_num_training_steps(experiment_config, 16)
    assert num_training_steps == 300

    experiment_config = {
        "searcher": {"max_length": {"records": 3000}},
    }
    num_training_steps = hf.compute_num_training_steps(experiment_config, 16)
    assert num_training_steps == 187

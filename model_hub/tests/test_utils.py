import os

import numpy as np

from model_hub import utils


def test_compute_num_training_steps() -> None:
    experiment_config = {"searcher": {"max_length": {"epochs": 3}}, "records_per_epoch": 124}
    num_training_steps = utils.compute_num_training_steps(experiment_config, 16)
    assert num_training_steps == 21

    experiment_config = {
        "searcher": {"max_length": {"batches": 300}},
    }
    num_training_steps = utils.compute_num_training_steps(experiment_config, 16)
    assert num_training_steps == 300

    experiment_config = {
        "searcher": {"max_length": {"records": 3000}},
    }
    num_training_steps = utils.compute_num_training_steps(experiment_config, 16)
    assert num_training_steps == 187


def test_expand_like() -> None:
    array_list = [np.array([[1, 2], [3, 4]]), np.array([[2, 3, 4], [3, 4, 5]])]
    result = utils.expand_like(array_list)
    assert np.array_equal(result, np.array([[1, 2, -100], [3, 4, -100], [2, 3, 4], [3, 4, 5]]))


def test_download_url() -> None:
    url = "https://images.freeimages.com/images/large-previews/5c6/sunset-jungle-1383333.jpg"
    file_path = utils.download_url("/tmp", url)
    assert os.path.exists(file_path)

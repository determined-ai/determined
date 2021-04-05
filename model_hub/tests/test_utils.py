import attrdict
import numpy as np

import model_hub.huggingface as hf
from model_hub import utils


def test_expand_like() -> None:
    array_list = [np.array([[1, 2], [3, 4]]), np.array([[2, 3, 4], [3, 4, 5]])]
    result = utils.expand_like(array_list)
    assert np.array_equal(result, np.array([[1, 2, -100], [3, 4, -100], [2, 3, 4], [3, 4, 5]]))


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


def test_config_parser() -> None:
    args = {"pretrained_model_name_or_path": "xnli", "num_labels": 4}
    config = hf.parse_dict_to_dataclasses((hf.ConfigKwargs,), args, as_dict=True)[0]
    target = attrdict.AttrDict(
        {
            "pretrained_model_name_or_path": "xnli",
            "revision": "main",
            "use_auth_token": False,
            "cache_dir": None,
            "num_labels": 4,
        }
    )
    assert config == target


def test_nodefault_config_parser() -> None:
    args = {
        "pretrained_model_name_or_path": "xnli",
    }
    config = hf.parse_dict_to_dataclasses((hf.ConfigKwargs,), args, as_dict=True)[0]
    target = attrdict.AttrDict(
        {
            "pretrained_model_name_or_path": "xnli",
            "revision": "main",
            "use_auth_token": False,
            "cache_dir": None,
        }
    )
    assert config == target

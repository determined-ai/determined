import attrdict

import model_hub.huggingface as hf


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

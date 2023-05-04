import copy
import json
import os
from typing import Dict

from determined.experimental import client
from ruamel import yaml

"""Very simple wrapper script which writes the ds_config.json from the yaml file and then launches
the experiment."""


def get_config_dict_from_yaml_path(path: str) -> Dict[str, any]:
    config = yaml.YAML(typ="safe")
    with open(path, "r") as f:
        config_dict = config.load(f)
    return config_dict


if __name__ == "__main__":
    config_dict = get_config_dict_from_yaml_path("autotune_config.yaml")
    ds_config = config_dict["hyperparameters"]["ds_config"]
    assert config_dict["searcher"]["metric"] == ds_config["autotuning"]["metric"]
    with open("ds_config.json", "w") as f:
        # Hack for type key in config here.
        ds_config_type_fix = copy.deepcopy(ds_config)
        ds_config_type_fix["optimizer"]["type"] = ds_config_type_fix["optimizer"].pop("TYPE")
        json.dump(ds_config_type_fix, f)
    client.create_experiment(config_dict, ".")
    os.remove("ds_config.json")

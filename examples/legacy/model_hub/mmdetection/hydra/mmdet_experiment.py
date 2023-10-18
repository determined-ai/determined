import os

import hydra
from omegaconf import DictConfig, MissingMandatoryValue, OmegaConf

from determined.common.experimental import Determined

CONTEXT_DIR = os.getcwd()


def check_for_missing(cfg):
    if isinstance(cfg, dict):
        for k, item in cfg.items():
            if item == "???":
                raise MissingMandatoryValue(f"Missing mandatory value for {k}.")
            check_for_missing(item)
    elif isinstance(cfg, list):
        for item in cfg:
            check_for_missing(item)


@hydra.main(config_path="./configs", config_name="config")
def my_experiment(cfg: DictConfig) -> None:
    config = OmegaConf.to_container(cfg, resolve=True)
    # We use a helper function now to check for missing values.
    # In the next version of omegaconf, we will be able to check for missing values by
    # passing throw_on_missing to the OmegaConf.to_container call above.
    check_for_missing(config)

    master = Determined()
    exp = master.create_experiment(config, CONTEXT_DIR)
    exp.activate()


if __name__ == "__main__":
    my_experiment()

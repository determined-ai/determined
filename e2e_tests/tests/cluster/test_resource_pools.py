import pytest

from determined.common import util
from determined.experimental import client
from tests import config as conf


@pytest.mark.e2e_cpu
def test_default_pool_task_container_defaults() -> None:
    # This test assumes the default resource pool in master config has non-empty
    # `task_containers_default` -> `environment_variables` configuration, for example:
    #
    #  - pool_name: default
    #    task_container_defaults:
    #    environment_variables:
    #      - SOMEVAR=SOMEVAL
    determined_master = conf.make_master_url()
    d = client.Determined(determined_master)
    config_path = conf.fixtures_path("no_op/single-medium-train-step.yaml")
    e1 = d.create_experiment(
        config=config_path,
        model_dir=conf.fixtures_path("no_op"),
    )

    e1_config = e1.config

    assert len(e1_config["environment"]["environment_variables"]["cpu"]) > 0

    with open(config_path) as fin:
        config_text = fin.read()
    parsed_config = util.safe_load_yaml_with_exceptions(config_text)
    parsed_config["resources"] = {"resource_pool": e1_config["resources"]["resource_pool"]}

    e2 = d.create_experiment(
        config=parsed_config,
        model_dir=conf.fixtures_path("no_op"),
    )
    e2_config = e2.config

    assert (
        e1_config["environment"]["environment_variables"]["cpu"]
        == e2_config["environment"]["environment_variables"]["cpu"]
    )

import pytest

from tests import api_utils
from tests.experiment import noop


@pytest.mark.e2e_cpu
def test_default_pool_task_container_defaults() -> None:
    # This test assumes the default resource pool in master config has non-empty
    # `task_containers_default` -> `environment_variables` configuration, for example:
    #
    #  - pool_name: default
    #    task_container_defaults:
    #    environment_variables:
    #      - SOMEVAR=SOMEVAL
    sess = api_utils.user_session()
    e1 = noop.create_experiment(sess)
    assert e1.config
    assert len(e1.config["environment"]["environment_variables"]["cpu"]) > 0

    config = {"resources": {"resource_pool": e1.config["resources"]["resource_pool"]}}
    e2 = noop.create_experiment(sess, config=config)
    assert e2.config
    assert (
        e1.config["environment"]["environment_variables"]["cpu"]
        == e2.config["environment"]["environment_variables"]["cpu"]
    )
    e1.kill()
    e2.kill()
    e1.wait(interval=0.01)
    e2.wait(interval=0.01)

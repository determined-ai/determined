import pytest

from determined.common.api.bindings import experimentv1State
from tests import config as conf
from tests import experiment as exp
from tests.api_utils import ADMIN_CREDENTIALS
from tests.cluster.test_users import logged_in_user


@pytest.mark.port_registry
def test_multi_trial_exp_port_registry() -> None:
    logged_in_user(ADMIN_CREDENTIALS)
    experiment_id = exp.create_experiment(
        conf.tutorials_path("mnist_pytorch/dist_random.yaml"),
        conf.tutorials_path("mnist_pytorch"),
    )

    exp.wait_for_experiment_state(
        experiment_id=experiment_id, target_state=experimentv1State.COMPLETED
    )

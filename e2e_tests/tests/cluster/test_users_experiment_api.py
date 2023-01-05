import pytest

from determined.common.api import bindings
from determined.experimental import client
from tests import config as conf
from tests import experiment as exp
from tests.cluster import test_users


@pytest.mark.e2e_cpu
def test_experiment_api_determined_disabled() -> None:
    test_users.log_in_user(test_users.ADMIN_CREDENTIALS)

    determined_master = conf.make_master_url()
    user_creds = test_users.create_test_user(add_password=True)

    test_users.det_spawn(["user", "deactivate", "determined"])
    try:
        d = client.Determined(determined_master, user_creds.username, user_creds.password)
        e = d.create_experiment(
            config=conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            model_dir=conf.fixtures_path("no_op"),
        )
        exp.wait_for_experiment_state(e.id, bindings.determinedexperimentv1State.STATE_COMPLETED)
    finally:
        test_users.det_spawn(["user", "activate", "determined"])

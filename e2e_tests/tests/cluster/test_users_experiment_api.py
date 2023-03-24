import pytest

from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils
from tests import config as conf
from tests import experiment as exp
from tests.cluster import test_users


@pytest.mark.e2e_cpu
def test_experiment_api_determined_disabled() -> None:
    api_utils.configure_token_store(test_users.ADMIN_CREDENTIALS)

    determined_master = conf.make_master_url()
    user_creds = api_utils.create_test_user(add_password=True)

    test_users.det_spawn(["user", "deactivate", "determined"])
    try:
        d = client.Determined(determined_master, user_creds.username, user_creds.password)
        e = d.create_experiment(
            config=conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            model_dir=conf.fixtures_path("no_op"),
        )
        exp.wait_for_experiment_state(e.id, bindings.experimentv1State.STATE_COMPLETED)
    finally:
        test_users.det_spawn(["user", "activate", "determined"])

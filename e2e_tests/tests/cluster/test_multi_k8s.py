import pytest

from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import experiment as exp

@pytest.mark.e2e_multi_k8s
def test_run_experiment_multi_k8s() -> None:
    sess = api_utils.user_session()
    exp_id = exp.create_experiment(
        sess,
        conf.fixtures_path("no_op/single-one-short-step.yaml"),
        conf.fixtures_path("no_op"),
        None,
    )
    exp.wait_for_experiment_state(
        sess, exp_id, bindings.experimentv1State.COMPLETED, max_wait_secs=500,
    )

    # TODO test out like the whole submitting into resource pools and all.
    # and getting boucnhed and needed to specify resource managers.

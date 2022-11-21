import pathlib

import pytest

import determined.common.api as determined_api
import determined.common.api.certs as certs
from determined.common import context, util
from determined.common.api.bindings import determinedexperimentv1State as EXP_STATE
from tests import config as conf
from tests import experiment as exp

from .test_users import ADMIN_CREDENTIALS, create_test_user, det_spawn, log_in_user


@pytest.mark.e2e_cpu
def test_experimental_experiment_api_determined_disabled() -> None:
    log_in_user(ADMIN_CREDENTIALS)
    context_path = pathlib.Path(conf.fixtures_path("no_op"))
    model_def_path = pathlib.Path(conf.fixtures_path("no_op/single-medium-train-step.yaml"))

    model_context = context.read_legacy_context(context_path)

    with model_def_path.open("r") as fin:
        dai_experiment_config = util.safe_load_yaml_with_exceptions(fin)

    determined_master = conf.make_master_url()
    user_creds = create_test_user(add_password=True)

    try:
        det_spawn(["user", "deactivate", "determined"])
        certs.cli_cert = certs.default_load(
            master_url=determined_master,
        )
        determined_api.authentication.cli_auth = determined_api.authentication.Authentication(
            determined_master,
            requested_user=user_creds.username,
            password=user_creds.password,
            try_reauth=True,
            cert=certs.cli_cert,
        )

        exp_id = determined_api.experiment.create_experiment_and_follow_logs(
            master_url=determined_master,
            config=dai_experiment_config,
            model_context=model_context,
            template=None,
            additional_body_fields={},
            activate=True,
            follow_first_trial_logs=False,
        )

        exp.wait_for_experiment_state(exp_id, EXP_STATE.STATE_COMPLETED)
    finally:
        det_spawn(["user", "activate", "determined"])

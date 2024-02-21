import sys

import pytest

from determined.common import api
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


def run_test_case(sess: api.Session, testcase: str, message: str) -> None:
    experiment_id = exp.create_experiment(
        sess,
        conf.fixtures_path(testcase),
        conf.fixtures_path("hpc"),
    )

    try:
        exp.wait_for_experiment_state(
            sess, experiment_id, bindings.experimentv1State.COMPLETED, max_wait_secs=600
        )

        trials = exp.experiment_trials(sess, experiment_id)

        assert exp.check_if_string_present_in_trial_logs(
            sess,
            trials[0].trial.id,
            message,
        )
    except AssertionError:
        # On failure print the log for triage
        logs = exp.trial_logs(sess, trials[0].trial.id, follow=True)
        tid = trials[0].trial.id
        print(f"******** Start of logs for trial {tid} ********", file=sys.stderr)
        print("".join(logs), file=sys.stderr)
        print(f"******** End of logs for trial {tid} ********", file=sys.stderr)
        print(
            f"Trial {tid} log did not contain any of the expected message: {message}",
            file=sys.stderr,
        )
        raise


# This test should succeed with Slurm plus all container types
# it does not yet succeed with PBS+Singularity.
@pytest.mark.e2e_slurm
def test_launch_embedded_quotes() -> None:
    sess = api_utils.user_session()
    run_test_case(
        sess,
        conf.fixtures_path("hpc/embedded-quotes.yaml"),
        'DATA: user_defined_key=datakey="datavalue with embedded "',
    )


@pytest.mark.e2e_slurm
def test_launch_embedded_single_quote() -> None:
    sess = api_utils.user_session()
    run_test_case(
        sess,
        conf.fixtures_path("hpc/embedded-single-quote.yaml"),
        'DATA: user_defined_key=datakey="datavalue with \' embedded "',
    )

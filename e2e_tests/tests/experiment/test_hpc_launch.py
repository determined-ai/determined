import sys
from typing import Callable

import pytest

from determined.common.api.bindings import experimentv1State
from tests import config as conf
from tests import experiment as exp


def run_test_case(testcase: str, message: str) -> None:
    experiment_id = exp.create_experiment(
        conf.fixtures_path(testcase),
        conf.fixtures_path("hpc"),
    )

    try:
        exp.wait_for_experiment_state(experiment_id, experimentv1State.COMPLETED, max_wait_secs=600)

        trials = exp.experiment_trials(experiment_id)

        assert exp.check_if_string_present_in_trial_logs(
            trials[0].trial.id,
            message,
        )
    except AssertionError:
        # On failure print the log for triage
        logs = exp.trial_logs(trials[0].trial.id, follow=True)
        print(
            "******** Start of logs for trial {} ********".format(trials[0].trial.id),
            file=sys.stderr,
        )
        print("".join(logs), file=sys.stderr)
        print(
            "******** End of logs for trial {} ********".format(trials[0].trial.id), file=sys.stderr
        )
        print(
            f"Trial {trials[0].trial.id} log did not contain any of the "
            + f"expected message: {message}",
            file=sys.stderr,
        )
        raise


# This test should succeed with Slurm plus all container types
# it does not yet succeed with PBS+Singularity.
@pytest.mark.e2e_slurm
def test_launch_embedded_quotes(collect_trial_profiles: Callable[[int], None]) -> None:
    run_test_case(
        conf.fixtures_path("hpc/embedded-quotes.yaml"),
        'DATA: user_defined_key=datakey="datavalue with embedded "',
    )


@pytest.mark.e2e_slurm
def test_launch_embedded_single_quote(collect_trial_profiles: Callable[[int], None]) -> None:
    run_test_case(
        conf.fixtures_path("hpc/embedded-single-quote.yaml"),
        'DATA: user_defined_key=datakey="datavalue with \' embedded "',
    )

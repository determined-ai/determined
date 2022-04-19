import subprocess

import pytest
import os

from determined.common.api import authentication, bindings, certs
from determined.common.experimental import session
from determined.common.api.bindings import determinedexperimentv1State
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_support_bundle():
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-one-short-step.yaml"), conf.fixtures_path("no_op")
    )

    exp.wait_for_experiment_state(exp_id, determinedexperimentv1State.STATE_COMPLETED)

    trial_id = exp.first_trial_in_experiment(exp_id)
    output_dir =  f"e2etest_trial_{trial_id}"
    os.mkdir(output_dir)

    command = ["det", "trial", "support-bundle", str(trial_id), "-o",output_dir]

    completed_process = subprocess.run(
        command, universal_newlines=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )

    assert completed_process.returncode == 0, "\nstdout:\n{} \nstderr:\n{}".format(
        completed_process.stdout, completed_process.stderr
    )

import os
import subprocess

import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_support_bundle() -> None:
    exp_id = exp.run_basic_test(
        config_file=conf.fixtures_path("no_op/single-one-short-step.yaml"),
        model_def_file=conf.fixtures_path("no_op"),
        expected_trials=1,
    )

    trial_id = exp.experiment_first_trial(exp_id)
    output_dir = f"e2etest_trial_{trial_id}"
    os.mkdir(output_dir)

    command = ["det", "trial", "support-bundle", str(trial_id), "-o", output_dir]

    completed_process = subprocess.run(
        command, universal_newlines=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )

    assert completed_process.returncode == 0, "\nstdout:\n{} \nstderr:\n{}".format(
        completed_process.stdout, completed_process.stderr
    )

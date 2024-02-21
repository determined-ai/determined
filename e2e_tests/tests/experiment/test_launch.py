from typing import Callable

import pytest

from determined.experimental import client
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
def test_launch_layer_mnist(collect_trial_profiles: Callable[[int], None]) -> None:
    sess = api_utils.user_session()
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 1)
    config = conf.set_profiling_enabled(config)
    config = conf.set_entrypoint(
        config, "python3 -m determined.launch.horovod --autohorovod python3 train.py"
    )

    experiment_id = exp.run_basic_test_with_temp_config(
        sess, config, conf.tutorials_path("mnist_pytorch"), 1
    )
    trials = exp.experiment_trials(sess, experiment_id)
    collect_trial_profiles(trials[0].trial.id)

    assert exp.check_if_string_present_in_trial_logs(
        sess,
        trials[0].trial.id,
        "resources exited successfully with a zero exit code",
    )


@pytest.mark.e2e_cpu
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
def test_launch_layer_exit(collect_trial_profiles: Callable[[int], None]) -> None:
    sess = api_utils.user_session()
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config = conf.set_entrypoint(config, "python3 -m nonexistent_launch_module python3 train.py")
    config["max_restarts"] = 0

    experiment_id = exp.run_failure_test_with_temp_config(
        sess, config, conf.tutorials_path("mnist_pytorch")
    )
    trials = exp.experiment_trials(sess, experiment_id)
    client.Determined._from_session(sess).get_trial(trials[0].trial.id)

    collect_trial_profiles(trials[0].trial.id)

    slurm_run = exp.check_if_string_present_in_trial_logs(
        sess, trials[0].trial.id, "Exited with exit code 1"
    )
    pbs_run = exp.check_if_string_present_in_trial_logs(
        sess, trials[0].trial.id, "exited with status 1"
    )
    cpu_run = exp.check_if_string_present_in_trial_logs(
        sess, trials[0].trial.id, "container failed with non-zero exit code: 1"
    )

    assert cpu_run or slurm_run or pbs_run

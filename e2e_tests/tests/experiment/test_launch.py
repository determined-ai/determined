from typing import Callable

import pytest

from determined.experimental import Determined
from tests import config as conf
from tests import experiment as exp


@pytest.mark.parallel
def test_launch_layer_cifar(collect_trial_profiles: Callable[[int], None]) -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 1)
    config = conf.set_profiling_enabled(config)
    config = conf.set_entrypoint(
        config, "python3 -m determined.launch.horovod --autohorovod --trial model_def:CIFARTrial"
    )

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_pytorch"), 1
    )
    trials = exp.experiment_trials(experiment_id)
    (
        Determined(conf.make_master_url())
        .get_trial(trials[0].trial.id)
        .select_checkpoint(latest=True)
        .load(map_location="cpu")
    )

    collect_trial_profiles(trials[0].trial.id)

    assert exp.check_if_string_present_in_trial_logs(
        trials[0].trial.id,
        "allocation stopped after resources exited successfully with a zero exit code",
    )


@pytest.mark.e2e_cpu
def test_launch_layer_exit(collect_trial_profiles: Callable[[int], None]) -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_pytorch/const.yaml"))
    config = conf.set_entrypoint(
        config, "python3 -m nonexistent_launch_module model_def:CIFARTrial"
    )

    experiment_id = exp.run_failure_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_pytorch")
    )
    trials = exp.experiment_trials(experiment_id)
    Determined(conf.make_master_url()).get_trial(trials[0].trial.id)

    collect_trial_profiles(trials[0].trial.id)

    assert exp.check_if_string_present_in_trial_logs(
        trials[0].trial.id, "container failed with non-zero exit code: 1"
    )

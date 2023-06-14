from typing import Callable, List

import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.gpu_required
@pytest.mark.distributed
@pytest.mark.parametrize("api_style", ["apex", "auto", "manual"])
def test_pytorch_distributed_with_amp(
    api_style: str, collect_trial_profiles: Callable[[int], None]
) -> None:
    config = conf.load_config(conf.fixtures_path(f"pytorch_amp/{api_style}_amp_distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_profiling_enabled(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.fixtures_path("pytorch_amp"), 1
    )
    trial_id = exp.experiment_trials(experiment_id)[0].trial.id
    collect_trial_profiles(trial_id)

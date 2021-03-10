import pytest

from determined.experimental import Determined
from tests import config as conf
from tests import experiment as exp

@pytest.mark.parallel  # type: ignore
def test_pl_mnist() -> None:
    exp_dir = "_mnist_pl"
    config = conf.load_config(conf.cv_examples_path(exp_dir + "/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path(exp_dir), 1
    )
    trials = exp.experiment_trials(experiment_id)
    (
        Determined(conf.make_master_url())
        .get_trial(trials[0]["id"])
        .select_checkpoint(latest=True)
        .load(map_location="cpu")
    )

@pytest.mark.parallel  # type: ignore
def test_pl_mnist_gan() -> None:
    exp_dir = "_gan_mnist_pl"
    config = conf.load_config(conf.gan_examples_path(exp_dir + "/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.gan_examples_path(exp_dir), 1
    )
    trials = exp.experiment_trials(experiment_id)
    (
        Determined(conf.make_master_url())
        .get_trial(trials[0]["id"])
        .select_checkpoint(latest=True)
        .load(map_location="cpu")
    )

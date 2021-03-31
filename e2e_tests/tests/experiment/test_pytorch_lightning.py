import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu  # type: ignore
def test_pl_mnist() -> None:
    exp_dir = "mnist_pl"
    config = conf.load_config(conf.cv_examples_path(exp_dir + "/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path(exp_dir), 1)


@pytest.mark.e2e_cpu  # type: ignore
def test_pl_mnist_gan() -> None:
    exp_dir = "gan_mnist_pl"
    config = conf.load_config(conf.gan_examples_path(exp_dir + "/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.gan_examples_path(exp_dir), 1)


@pytest.mark.e2e_gpu  # type: ignore
@pytest.mark.parametrize("api_style", ["apex", "auto"])  # type: ignore
def test_pl_const_with_amp(api_style: str) -> None:
    dir_name = "pytorch_lightning_amp"
    config = conf.load_config(conf.fixtures_path(dir_name + "/" + api_style + "_amp.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.fixtures_path(dir_name), 1)

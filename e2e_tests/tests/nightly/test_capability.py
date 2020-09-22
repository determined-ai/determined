import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.nightly  # type: ignore
def test_nas_search() -> None:
    config = conf.load_config(conf.experimental_path("trial/rsws_nas/train_one_arch.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.experimental_path("trial/rsws_nas"), 1)


@pytest.mark.nightly  # type: ignore
def test_bert_glue() -> None:
    config = conf.load_config(conf.experimental_path("trial/bert_glue_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/bert_glue_pytorch/"), 1
    )


@pytest.mark.nightly  # type: ignore
def test_resnet50() -> None:
    config = conf.load_config(conf.experimental_path("trial/resnet50_tf_keras/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/resnet50_tf_keras"), 1
    )


@pytest.mark.nightly  # type: ignore
def test_mnist_pytorch_multi_output() -> None:
    config = conf.load_config(conf.experimental_path("trial/mnist_pytorch_multi_output/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/mnist_pytorch_multi_output"), 1
    )


@pytest.mark.nightly  # type: ignore
def test_imagenet_nas() -> None:
    config = conf.load_config(conf.experimental_path("trial/gaea_nas/eval/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.experimental_path("trial/gaea_nas/eval"), 1)


@pytest.mark.nightly  # type: ignore
def test_gbt_estimator() -> None:
    config = conf.load_config(conf.experimental_path("trial/gbt_estimator/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.experimental_path("trial/gbt_estimator"), 1)

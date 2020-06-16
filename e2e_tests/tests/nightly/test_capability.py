import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.nightly  # type: ignore
def test_nas_search() -> None:
    config = conf.load_config(conf.experimental_path("trial/nas_search/train_one_arch.yaml"))
    config = conf.set_max_steps(config, 2)

    exp.run_basic_test_with_temp_config(config, conf.experimental_path("trial/nas_search"), 1)


@pytest.mark.nightly  # type: ignore
def test_bert_glue() -> None:
    config = conf.load_config(conf.experimental_path("trial/bert_glue_pytorch/const.yaml"))
    config = conf.set_max_steps(config, 2)

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/bert_glue_pytorch/"), 1
    )


@pytest.mark.nightly  # type: ignore
def test_faster_rcnn() -> None:
    config = conf.load_config(conf.experimental_path("trial/FasterRCNN_tp/16-gpus.yaml"))
    config = conf.set_max_steps(config, 2)
    config = conf.set_slots_per_trial(config, 1)

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/FasterRCNN_tp"), 1, max_wait_secs=4800
    )


@pytest.mark.nightly  # type: ignore
def test_mnist_tp_to_estimator() -> None:
    config = conf.load_config(conf.experimental_path("trial/mnist_tp_to_estimator/const.yaml"))
    config = conf.set_max_steps(config, 2)

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/mnist_tp_to_estimator"), 1
    )


@pytest.mark.nightly  # type: ignore
def test_resnet50() -> None:
    config = conf.load_config(conf.experimental_path("trial/resnet50_tf_keras/const.yaml"))
    config = conf.set_max_steps(config, 2)

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/resnet50_tf_keras"), 1
    )


@pytest.mark.nightly  # type: ignore
def test_mnist_pytorch_multi_output() -> None:
    config = conf.load_config(conf.experimental_path("trial/mnist_pytorch_multi_output/const.yaml"))
    config = conf.set_max_steps(config, 2)

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/mnist_pytorch_multi_output"), 1
    )


@pytest.mark.nightly  # type: ignore
def test_imagenet_nas() -> None:
    config = conf.load_config(conf.experimental_path("trial/imagenet_nas_arch_pytorch/const.yaml"))
    config = conf.set_max_steps(config, 2)

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/imagenet_nas_arch_pytorch"), 1
    )


@pytest.mark.nightly  # type: ignore
def test_gbt_estimator() -> None:
    config = conf.load_config(conf.experimental_path("trial/gbt_estimator/const.yaml"))
    config = conf.set_max_steps(config, 2)

    exp.run_basic_test_with_temp_config(config, conf.experimental_path("trial/gbt_estimator"), 1)

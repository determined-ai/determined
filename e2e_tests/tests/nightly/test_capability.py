import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.nightly  # type: ignore
def test_mmdetection_pytorch_const() -> None:
    config = conf.load_config(conf.cv_examples_path("mmdetection_pytorch/const_fake_data.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("mmdetection_pytorch"), 1)


@pytest.mark.nightly  # type: ignore
def test_bert_glue_const() -> None:
    config = conf.load_config(conf.nlp_examples_path("bert_glue_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.nlp_examples_path("bert_glue_pytorch"), 1)


@pytest.mark.nightly  # type: ignore
def test_gaea_pytorch_const() -> None:
    config = conf.load_config(conf.nas_examples_path("gaea_pytorch/eval/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.nas_examples_path("gaea_pytorch/eval"), 1)


@pytest.mark.nightly  # type: ignore
def test_protonet_omniglot_pytorch_const() -> None:
    config = conf.load_config(conf.meta_learning_examples_path("protonet_omniglot_pytorch/20way1shot.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.meta_learning_examples_path("protonet_omniglot_pytorch"), 1)


@pytest.mark.nightly  # type: ignore
def test_gan_mnist_pytorch_const() -> None:
    config = conf.load_config(conf.gan_examples_path("gan_mnist_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.gan_examples_path("gan_mnist_pytorch"), 1)

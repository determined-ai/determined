import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.distributed  # type: ignore
def test_mnist_pytorch_distributed() -> None:
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.tutorials_path("mnist_pytorch"), 1)


@pytest.mark.distributed  # type: ignore
def test_fashion_mnist_tf_keras_distributed() -> None:
    config = conf.load_config(conf.tutorials_path("fashion_mnist_tf_keras/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.tutorials_path("fashion_mnist_tf_keras"), 1)


@pytest.mark.distributed  # type: ignore
def test_cifar10_pytorch_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_pytorch/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("cifar10_pytorch"), 1)


@pytest.mark.distributed  # type: ignore
def test_mnist_estimator_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("mnist_estimator/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("mnist_estimator"), 1)


@pytest.mark.distributed  # type: ignore
def test_cifar10_tf_keras_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_tf_keras/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("cifar10_tf_keras"), 1)


@pytest.mark.distributed  # type: ignore
def test_iris_tf_keras_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("iris_tf_keras/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("iris_tf_keras"), 1)


@pytest.mark.distributed  # type: ignore
def test_unets_tf_keras_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("unets_tf_keras/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("unets_tf_keras"), 1)


@pytest.mark.distributed  # type: ignore
def test_bert_glue_pytorch_distributed() -> None:
    config = conf.load_config(conf.nlp_examples_path("bert_glue_pytorch/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.nlp_examples_path("bert_glue_pytorch"), 1)


@pytest.mark.distributed  # type: ignore
def test_gaea_pytorch_distributed() -> None:
    config = conf.load_config(
        conf.nas_examples_path("gaea_pytorch/eval/distributed_no_data_download.yaml")
    )
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.nas_examples_path("gaea_pytorch/eval"), 1)


@pytest.mark.distributed  # type: ignore
def test_gan_mnist_pytorch_distributed() -> None:
    config = conf.load_config(conf.gan_examples_path("gan_mnist_pytorch/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.gan_examples_path("gan_mnist_pytorch"), 1)

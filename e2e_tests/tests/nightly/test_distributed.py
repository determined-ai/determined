import os
import shutil
import tempfile
from typing import Generator

import pytest
from compute_stats import compare_stats

from tests import config as conf
from tests import experiment as exp


@pytest.mark.distributed
def test_mnist_pytorch_distributed() -> None:
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.tutorials_path("mnist_pytorch"), 1)


@pytest.mark.distributed
def test_fashion_mnist_tf_keras_distributed() -> None:
    config = conf.load_config(conf.tutorials_path("fashion_mnist_tf_keras/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.tutorials_path("fashion_mnist_tf_keras"), 1)


@pytest.mark.distributed
def test_imagenet_pytorch_distributed() -> None:
    config = conf.load_config(conf.tutorials_path("imagenet_pytorch/distributed_cifar.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.tutorials_path("imagenet_pytorch"), 1)


@pytest.mark.distributed
def test_cifar10_pytorch_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_pytorch/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("cifar10_pytorch"), 1)


@pytest.mark.distributed
@pytest.mark.gpu_required
def test_mmdetection_pytorch_distributed() -> None:
    config = conf.load_config(
        conf.cv_examples_path("mmdetection_pytorch/distributed_fake_data.yaml")
    )
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("mmdetection_pytorch"), 1)


@pytest.mark.distributed
def test_mnist_estimator_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("mnist_estimator/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("mnist_estimator"), 1)


@pytest.mark.distributed
def test_cifar10_tf_keras_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_tf_keras/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("cifar10_tf_keras"), 1)


@pytest.mark.distributed
def test_iris_tf_keras_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("iris_tf_keras/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("iris_tf_keras"), 1)


@pytest.mark.distributed
@pytest.mark.gpu_required
def test_unets_tf_keras_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("unets_tf_keras/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    download_dir = "/tmp/data"
    url = "https://s3-us-west-2.amazonaws.com/determined-ai-datasets/oxford_iiit_pet/oxford_iiit_pet.tar.gz"  # noqa

    with tempfile.TemporaryDirectory() as tmpdir:
        copy_destination = os.path.join(tmpdir, "example")
        shutil.copytree(conf.cv_examples_path("unets_tf_keras"), copy_destination)
        with open(os.path.join(copy_destination, "startup-hook.sh"), "a") as f:
            f.write("\n")
            f.write(f"wget -O /tmp/data.tar.gz {url}\n")
            f.write(f"mkdir {download_dir}\n")
            f.write(f"tar -xzvf /tmp/data.tar.gz -C {download_dir}\n")
        exp.run_basic_test_with_temp_config(config, copy_destination, 1)


@pytest.mark.distributed
def test_bert_glue_pytorch_distributed() -> None:
    config = conf.load_config(conf.nlp_examples_path("bert_glue_pytorch/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.nlp_examples_path("bert_glue_pytorch"), 1)


@pytest.mark.distributed
@pytest.mark.gpu_required
def test_gaea_pytorch_distributed() -> None:
    config = conf.load_config(
        conf.nas_examples_path("gaea_pytorch/eval/distributed_no_data_download.yaml")
    )
    config = conf.set_global_batch_size(config, 256)
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.nas_examples_path("gaea_pytorch/eval"), 1)


@pytest.mark.distributed
def test_gan_mnist_pytorch_distributed() -> None:
    config = conf.load_config(conf.gan_examples_path("gan_mnist_pytorch/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.gan_examples_path("gan_mnist_pytorch"), 1)


@pytest.mark.distributed
@pytest.mark.gpu_required
def test_detr_coco_pytorch_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("detr_coco_pytorch/const_fake.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 2)

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("detr_coco_pytorch"), 1)


@pytest.mark.distributed
@pytest.mark.gpu_required
def test_deformabledetr_coco_pytorch_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("deformabledetr_coco_pytorch/const_fake.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_global_batch_size(config, 2)
    config = conf.set_slots_per_trial(config, 2)

    exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("deformabledetr_coco_pytorch"), 1
    )


@pytest.mark.distributed
def test_word_language_transformer_distributed() -> None:
    config = conf.load_config(conf.nlp_examples_path("word_language_model/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = config.copy()
    config["hyperparameters"]["model_cls"] = "Transformer"

    exp.run_basic_test_with_temp_config(config, conf.nlp_examples_path("word_language_model"), 1)


@pytest.mark.distributed
def test_word_language_lstm_const() -> None:
    config = conf.load_config(conf.nlp_examples_path("word_language_model/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = config.copy()
    config["hyperparameters"]["model_cls"] = "LSTM"
    config["hyperparameters"]["tied"] = False

    exp.run_basic_test_with_temp_config(config, conf.nlp_examples_path("word_language_model"), 1)


@pytest.mark.distributed
@pytest.mark.gpu_required
def test_byol_pytorch_distributed() -> None:
    config = conf.load_config(conf.cv_examples_path("byol_pytorch/distributed-stl10.yaml"))
    config = conf.set_max_length(config, {"epochs": 1})

    exp.run_basic_test_with_temp_config(config, conf.cv_examples_path("byol_pytorch"), 1)


@pytest.fixture(scope="session", autouse=True)
def resource_stats() -> Generator[None, None, None]:
    yield
    compare_stats()

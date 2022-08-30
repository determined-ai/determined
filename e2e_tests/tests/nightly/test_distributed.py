import os
import shutil
import tempfile

import pytest

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
def test_pix2pix_facades_distributed() -> None:
    config = conf.load_config(conf.gan_examples_path("pix2pix_tf_keras/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.gan_examples_path("pix2pix_tf_keras"), 1)


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


@pytest.mark.deepspeed
@pytest.mark.gpu_required
def test_deepspeed_moe() -> None:
    config = conf.load_config(conf.deepspeed_examples_path("cifar10_moe/moe.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.deepspeed_examples_path("cifar10_moe"), 1)


@pytest.mark.deepspeed
@pytest.mark.gpu_required
def test_deepspeed_zero() -> None:
    config = conf.load_config(conf.deepspeed_examples_path("cifar10_moe/zero_stages.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_min_validation_period(config, {"batches": 100})

    exp.run_basic_test_with_temp_config(config, conf.deepspeed_examples_path("cifar10_moe"), 1)


@pytest.mark.deepspeed
@pytest.mark.gpu_required
def test_deepspeed_pipeline_parallel() -> None:
    config = conf.load_config(conf.deepspeed_examples_path("pipeline_parallelism/distributed.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_min_validation_period(config, {"batches": 100})

    exp.run_basic_test_with_temp_config(
        config, conf.deepspeed_examples_path("pipeline_parallelism"), 1
    )


@pytest.mark.deepspeed
@pytest.mark.gpu_required
def test_gpt_neox_zero_medium() -> None:
    config = conf.load_config(conf.deepspeed_examples_path("gpt_neox/zero3_medium.yaml"))
    config = conf.set_max_length(config, {"batches": 100})
    config = conf.set_min_validation_period(config, {"batches": 100})

    exp.run_basic_test_with_temp_config(config, conf.deepspeed_examples_path("gpt_neox"), 1)


@pytest.mark.deepspeed
@pytest.mark.gpu_required
def test_gpt_neox_zero_3D_parallel() -> None:
    config = conf.load_config(conf.deepspeed_examples_path("gpt_neox/zero1_3d_parallel.yaml"))
    config = conf.set_max_length(config, {"batches": 100})
    config = conf.set_min_validation_period(config, {"batches": 100})

    exp.run_basic_test_with_temp_config(config, conf.deepspeed_examples_path("gpt_neox"), 1)


@pytest.mark.deepspeed
@pytest.mark.gpu_required
def test_deepspeed_dcgan() -> None:
    config = conf.load_config(conf.deepspeed_examples_path("deepspeed_dcgan/mnist.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.deepspeed_examples_path("deepspeed_dcgan"), 1)


@pytest.mark.deepspeed
@pytest.mark.gpu_required
def test_deepspeed_cpu_offloading() -> None:
    config = conf.load_config(
        conf.deepspeed_examples_path("cifar10_cpu_offloading/zero_stages_3_offload.yaml")
    )
    config = conf.set_max_length(config, {"batches": 100})

    exp.run_basic_test_with_temp_config(
        config, conf.deepspeed_examples_path("cifar10_cpu_offloading"), 1
    )

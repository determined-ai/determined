import pathlib
import subprocess

import pytest

official_examples = [
    (
        "official/trial/cifar10_cnn_pytorch",
        "official/trial/cifar10_cnn_pytorch/const.yaml",
    ),
    (
        "official/trial/cifar10_cnn_tf_keras",
        "official/trial/cifar10_cnn_tf_keras/const.yaml",
    ),
    (
        "official/trial/fashion_mnist_tf_keras",
        "official/trial/fashion_mnist_tf_keras/const.yaml",
    ),
    ("official/trial/iris_tf_keras", "official/trial/iris_tf_keras/const.yaml"),
    ("official/trial/mnist_estimator", "official/trial/mnist_estimator/const.yaml"),
    ("official/trial/mnist_pytorch", "official/trial/mnist_pytorch/const.yaml"),
    (
        "official/trial/multiple_lr_schedulers_pytorch",
        "official/trial/multiple_lr_schedulers_pytorch/const.yaml",
    ),
    ("official/trial/mnist_gan_pytorch", "official/trial/mnist_gan_pytorch/const.yaml"),
    # TODO(DET-2931): A full validation step in this example is too expensive
    # to run this test in under a few minutes. Add it back in once we can test
    # a single batch of validation.
    # ("official/object_detection_pytorch", "official/object_detection_pytorch/const.yaml"),
]


@pytest.mark.parametrize("model_def,config_file", official_examples)
def test_official(model_def: str, config_file: str) -> None:
    examples_dir = pathlib.Path(__file__).parent.parent
    model_def_absolute = examples_dir.joinpath(model_def)
    config_file_absolute = examples_dir.joinpath(config_file)

    subprocess.check_output(
        (
            "det",
            "experiment",
            "create",
            "--local",
            "--test",
            str(config_file_absolute),
            str(model_def_absolute),
        )
    )

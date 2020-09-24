import pathlib
import subprocess

import pytest

official_examples = [
    (
        "computer_vision/cifar10_pytorch",
        "computer_vision/cifar10_pytorch/const.yaml",
    ),
    (
        "computer_vision/cifar10_tf_keras",
        "computer_vision/cifar10_tf_keras/const.yaml",
    ),
    (
        "tutorials/fashion_mnist_tf_keras",
        "tutorials/fashion_mnist_tf_keras/const.yaml",
    ),
    ("computer_vision/iris_tf_keras", "computer_vision/iris_tf_keras/const.yaml"),
    ("computer_vision/mnist_estimator", "computer_vision/mnist_estimator/const.yaml"),
    ("tutorials/mnist_pytorch", "tutorials/mnist_pytorch/const.yaml"),
    ("gan/gan_mnist_pytorch", "gan/gan_mnist_pytorch/const.yaml"),
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

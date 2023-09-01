from typing import Any, Dict, List

import pytest

from determined.experimental import client as _client
from tests import config as conf
from tests import experiment as exp


def _get_validation_metrics(client: _client.Determined, trial_id: int) -> List[Dict[str, Any]]:
    return [m.metrics for m in client.stream_trials_validation_metrics([trial_id])]


@pytest.mark.nightly
def test_mnist_pytorch_accuracy(client: _client.Determined) -> None:
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("mnist_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    validations = _get_validation_metrics(client, trials[0].trial.id)
    validation_accuracies = [v["accuracy"] for v in validations]

    target_accuracy = 0.97
    assert max(validation_accuracies) > target_accuracy, (
        "mnist_pytorch did not reach minimum target accuracy {}."
        " full validation accuracy history: {}".format(target_accuracy, validation_accuracies)
    )


@pytest.mark.nightly
def test_fashion_mnist_tf_keras(client: _client.Determined) -> None:
    config = conf.load_config(conf.tutorials_path("fashion_mnist_tf_keras/const.yaml"))
    config = conf.set_random_seed(config, 1591110586)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("fashion_mnist_tf_keras"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    validations = _get_validation_metrics(client, trials[0].trial.id)
    validation_accuracies = [v["val_accuracy"] for v in validations]

    target_accuracy = 0.85
    assert max(validation_accuracies) > target_accuracy, (
        "fashion_mnist_tf_keras did not reach minimum target accuracy {}."
        " full validation accuracy history: {}".format(target_accuracy, validation_accuracies)
    )


@pytest.mark.nightly
def test_mnist_estimator_accuracy(client: _client.Determined) -> None:
    config = conf.load_config(conf.fixtures_path("mnist_estimator/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.fixtures_path("mnist_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    validations = _get_validation_metrics(client, trials[0].trial.id)

    validation_accuracies = [v["accuracy"] for v in validations]

    target_accuracy = 0.95
    assert max(validation_accuracies) > target_accuracy, (
        "mnist_estimator did not reach minimum target accuracy {}."
        " full validation accuracy history: {}".format(target_accuracy, validation_accuracies)
    )


@pytest.mark.nightly_quarantine
def test_cifar10_tf_keras_accuracy(client: _client.Determined) -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_tf_keras/const.yaml"))
    config = conf.set_random_seed(config, 1591110586)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_tf_keras"), 1, None, 6000
    )
    trials = exp.experiment_trials(experiment_id)
    validations = _get_validation_metrics(client, trials[0].trial.id)
    validation_accuracies = [v["val_categorical_accuracy"] for v in validations]

    target_accuracy = 0.73
    assert max(validation_accuracies) > target_accuracy, (
        "cifar10_pytorch did not reach minimum target accuracy {}."
        " full validation accuracy history: {}".format(target_accuracy, validation_accuracies)
    )

@pytest.mark.nightly
def test_hf_trainer_api_accuracy(client: _client.Determined) -> None:
    test_dir = "hf_image_classification"
    config = conf.load_config(conf.hf_trainer_examples_path(f"{test_dir}/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.hf_trainer_examples_path(test_dir), 1
    )

    trials = exp.experiment_trials(experiment_id)
    validations = _get_validation_metrics(client, trials[0].trial.id)
    validation_accuracies = [v["eval_accuracy"] for v in validations]

    target_accuracy = 0.82
    assert max(validation_accuracies) > target_accuracy, (
        "hf_trainer_api did not reach minimum target accuracy {}."
        " full validation accuracy history: {}".format(target_accuracy, validation_accuracies)
    )

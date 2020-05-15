import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.nightly  # type: ignore
def test_cifar10_pytorch_accuracy() -> None:
    config = conf.load_config(conf.official_examples_path("cifar10_cnn_pytorch/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("cifar10_cnn_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_errors = [
        step["validation"]["metrics"]["validation_metrics"]["validation_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.74
    assert max(validation_errors) > target_accuracy, (
        "cifar10_cnn_pytorch did not reach minimum target accuracy {} in {} steps."
        " full validation error history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_errors
        )
    )


@pytest.mark.nightly  # type: ignore
def test_mnist_tp_accuracy() -> None:
    config = conf.load_config(conf.official_examples_path("mnist_tp/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("mnist_tp"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_errors = []
    # TODO (DET-3082): The validation metric names were modified by our trial reporting
    # from accuracy to val_accuracy.  We should probably remove the added prefix so
    # the metric name is as specified.
    validation_errors = [
        step["validation"]["metrics"]["validation_metrics"]["val_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.95
    assert max(validation_errors) > target_accuracy, (
        "mnist_tp did not reach minimum target accuracy {} in {} steps."
        " full validation error history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_errors
        )
    )


@pytest.mark.nightly  # type: ignore
def test_mnist_estimator_accuracy() -> None:
    config = conf.load_config(conf.official_examples_path("mnist_estimator/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("mnist_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_errors = [
        step["validation"]["metrics"]["validation_metrics"]["accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.95
    assert max(validation_errors) > target_accuracy, (
        "mnist_estimator did not reach minimum target accuracy {} in {} steps."
        " full validation error history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_errors
        )
    )

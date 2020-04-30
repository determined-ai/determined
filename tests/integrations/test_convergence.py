import pytest

from tests.integrations import config as conf
from tests.integrations import experiment as exp


@pytest.mark.nightly  # type: ignore
def test_cifar10_pytorch_accuracy() -> None:
    config = conf.load_config(conf.official_examples_path("cifar10_cnn_pytorch/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("cifar10_cnn_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].id)

    validation_errors = [
        step.validation.metrics["validation_metrics"]["validation_accuracy"]
        for step in trial_metrics.steps
        if step.validation
    ]

    target_accuracy = 0.745
    assert max(validation_errors) > target_accuracy, (
        "cifar10_cnn_pytorch did not reach minimum target accuracy {} in {} steps."
        " full validation error history: {}".format(
            target_accuracy, len(trial_metrics.steps), validation_errors
        )
    )

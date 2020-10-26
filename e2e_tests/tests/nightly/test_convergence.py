import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.nightly  # type: ignore
def test_mnist_pytorch_accuracy() -> None:
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("mnist_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.97
    assert max(validation_accuracies) > target_accuracy, (
        "mnist_pytorch did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly  # type: ignore
def test_fashion_mnist_tf_keras() -> None:
    config = conf.load_config(conf.tutorials_path("fashion_mnist_tf_keras/const.yaml"))
    config = conf.set_random_seed(config, 1591110586)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("fashion_mnist_tf_keras"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["val_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.85
    assert max(validation_accuracies) > target_accuracy, (
        "fashion_mnist_tf_keras did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly  # type: ignore
def test_cifar10_pytorch_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_pytorch/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["validation_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.74
    assert max(validation_accuracies) > target_accuracy, (
        "cifar10_pytorch did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly  # type: ignore
def test_fasterrcnn_coco_pytorch_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("fasterrcnn_coco_pytorch/const.yaml"))
    config = conf.set_random_seed(config, 1590497309)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("fasterrcnn_coco_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_iou = [
        step["validation"]["metrics"]["validation_metrics"]["val_avg_iou"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_iou = 0.42
    assert max(validation_iou) > target_iou, (
        "fasterrcnn_coco_pytorch did not reach minimum target accuracy {} in {} steps."
        " full validation avg_iou history: {}".format(
            target_iou, len(trial_metrics["steps"]), validation_iou
        )
    )


@pytest.mark.nightly  # type: ignore
def test_mnist_estimator_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("mnist_estimator/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("mnist_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.95
    assert max(validation_accuracies) > target_accuracy, (
        "mnist_estimator did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly  # type: ignore
def test_mnist_tf_layers_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("mnist_tf_layers/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("mnist_tf_layers"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_errors = [
        step["validation"]["metrics"]["validation_metrics"]["error"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_error = 0.03
    assert min(validation_errors) < target_error, (
        "mnist_estimator did not reach minimum target error {} in {} steps."
        " full validation error history: {}".format(
            target_error, len(trial_metrics["steps"]), validation_errors
        )
    )


@pytest.mark.nightly  # type: ignore
def test_cifar10_tf_keras_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_tf_keras/const.yaml"))
    config = conf.set_random_seed(config, 1591110586)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_tf_keras"), 1, None, 6000
    )
    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["val_categorical_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.74
    assert max(validation_accuracies) > target_accuracy, (
        "cifar10_pytorch did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly  # type: ignore
def test_iris_tf_keras_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("iris_tf_keras/const.yaml"))
    config = conf.set_random_seed(config, 1591280374)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("iris_tf_keras"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["val_categorical_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.95
    assert max(validation_accuracies) > target_accuracy, (
        "iris_tf_keras did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly  # type: ignore
def test_unets_tf_keras_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("unets_tf_keras/const.yaml"))
    config = conf.set_random_seed(config, 1591280374)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("unets_tf_keras"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["val_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.85
    assert max(validation_accuracies) > target_accuracy, (
        "unets_tf_keras did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly  # type: ignore
def test_protonet_omniglot_pytorch_accuracy() -> None:
    config = conf.load_config(
        conf.meta_learning_examples_path("protonet_omniglot_pytorch/20way1shot.yaml")
    )
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.meta_learning_examples_path("protonet_omniglot_pytorch"), 1, None, 6000
    )
    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["acc"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.96
    assert max(validation_accuracies) > target_accuracy, (
        "protonet_omniglot_pytorch did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly  # type: ignore
def test_gan_mnist_pytorch_accuracy() -> None:
    config = conf.load_config(conf.gan_examples_path("gan_mnist_pytorch/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.gan_examples_path("gan_mnist_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_losses = [
        step["validation"]["metrics"]["validation_metrics"]["loss"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_loss = 0.40
    assert min(validation_losses) < target_loss, (
        "gan_mnist_pytorch did not reach minimum target loss {} in {} steps."
        " full validation loss history: {}".format(
            target_loss, len(trial_metrics["steps"]), validation_losses
        )
    )


@pytest.mark.nightly  # type: ignore
def test_gbt_titanic_estimator_accuracy() -> None:
    config = conf.load_config(conf.decision_trees_examples_path("gbt_titanic_estimator/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.decision_trees_examples_path("gbt_titanic_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.74
    assert max(validation_accuracies) > target_accuracy, (
        "gbt_titanic_estimator did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly  # type: ignore
def test_data_layer_mnist_estimator_accuracy() -> None:
    config = conf.load_config(
        conf.data_layer_examples_path("data_layer_mnist_estimator/const.yaml")
    )
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.data_layer_examples_path("data_layer_mnist_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.94
    assert max(validation_accuracies) > target_accuracy, (
        "data_layer_mnist_estimator did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly  # type: ignore
def test_data_layer_mnist_tf_keras_accuracy() -> None:
    config = conf.load_config(conf.data_layer_examples_path("data_layer_mnist_tf_keras/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.data_layer_examples_path("data_layer_mnist_tf_keras"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0]["id"])

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["val_sparse_categorical_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.97
    assert max(validation_accuracies) > target_accuracy, (
        "data_layer_mnist_tf_keras did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )

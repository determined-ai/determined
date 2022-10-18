import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.nightly
def test_mnist_pytorch_accuracy() -> None:
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("mnist_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

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


@pytest.mark.nightly
def test_fashion_mnist_tf_keras() -> None:
    config = conf.load_config(conf.tutorials_path("fashion_mnist_tf_keras/const.yaml"))
    config = conf.set_random_seed(config, 1591110586)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("fashion_mnist_tf_keras"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

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


@pytest.mark.nightly
def test_imagenet_pytorch() -> None:
    config = conf.load_config(conf.tutorials_path("imagenet_pytorch/const_cifar.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("imagenet_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

    validation_loss = [
        step["validation"]["metrics"]["validation_metrics"]["val_loss"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_loss = 1.55
    assert max(validation_loss) < target_loss, (
        "imagenet_pytorch did not reach minimum target loss {} in {} steps."
        " full validation accuracy history: {}".format(
            target_loss, len(trial_metrics["steps"]), validation_loss
        )
    )


@pytest.mark.nightly
def test_cifar10_pytorch_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_pytorch/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["validation_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.73
    assert max(validation_accuracies) > target_accuracy, (
        "cifar10_pytorch did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly
def test_fasterrcnn_coco_pytorch_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("fasterrcnn_coco_pytorch/const.yaml"))
    config = conf.set_random_seed(config, 1590497309)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("fasterrcnn_coco_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

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


@pytest.mark.nightly
def test_mnist_estimator_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("mnist_estimator/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("mnist_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

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


@pytest.mark.nightly
def test_cifar10_tf_keras_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("cifar10_tf_keras/const.yaml"))
    config = conf.set_random_seed(config, 1591110586)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("cifar10_tf_keras"), 1, None, 6000
    )
    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["val_categorical_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.73
    assert max(validation_accuracies) > target_accuracy, (
        "cifar10_pytorch did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly
def test_iris_tf_keras_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("iris_tf_keras/const.yaml"))
    config = conf.set_random_seed(config, 1591280374)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("iris_tf_keras"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

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


@pytest.mark.nightly
def test_unets_tf_keras_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("unets_tf_keras/const.yaml"))
    config = conf.set_random_seed(config, 1591280374)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("unets_tf_keras"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

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


@pytest.mark.nightly
def test_gbt_titanic_estimator_accuracy() -> None:
    config = conf.load_config(conf.decision_trees_examples_path("gbt_titanic_estimator/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.decision_trees_examples_path("gbt_titanic_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

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


@pytest.mark.nightly
def test_cifar10_byol_pytorch_accuracy() -> None:
    config = conf.load_config(conf.cv_examples_path("byol_pytorch/const-cifar10.yaml"))
    # Limit convergence time, since was running over 30 minute limit.
    config["searcher"]["max_length"]["epochs"] = 20
    config["hyperparameters"]["classifier"]["train_epochs"] = 1
    config = conf.set_random_seed(config, 1591280374)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("byol_pytorch"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["test_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    # Accuracy reachable within limited convergence time -- goes higher given full training.
    target_accuracy = 0.40
    assert max(validation_accuracies) > target_accuracy, (
        "cifar10_byol_pytorch did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )


@pytest.mark.nightly
def test_hf_trainer_api_accuracy() -> None:
    config = conf.load_config(conf.integrations_examples_path("hf_trainer_api/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.integrations_examples_path("hf_trainer_api"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    trial_metrics = exp.trial_metrics(trials[0].trial.id)

    validation_accuracies = [
        step["validation"]["metrics"]["validation_metrics"]["eval_accuracy"]
        for step in trial_metrics["steps"]
        if step.get("validation")
    ]

    target_accuracy = 0.90
    assert max(validation_accuracies) > target_accuracy, (
        "hf_trainer_api did not reach minimum target accuracy {} in {} steps."
        " full validation accuracy history: {}".format(
            target_accuracy, len(trial_metrics["steps"]), validation_accuracies
        )
    )

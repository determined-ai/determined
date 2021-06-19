import pathlib

import pytest
import tensorflow as tf

import determined as det
from determined import estimator, experimental, keras, pytorch
from tests.experiment.fixtures import (
    estimator_linear_model,
    pytorch_onevar_model,
    pytorch_xor_model,
    tf_keras_one_var_model,
)


def test_test_one_batch() -> None:
    with det._local_execution_manager(pathlib.Path(pytorch_xor_model.__file__).parent):
        experimental.test_one_batch(
            trial_class=pytorch_xor_model.XORTrial,
            config={
                "hyperparameters": {"hidden_size": 2, "learning_rate": 0.5, "global_batch_size": 4},
                "searcher": {"metric": "loss"},
            },
        )


def test_pytorch_from_config() -> None:
    config = {"hyperparameters": {"global_batch_size": 4}}
    context = pytorch.PyTorchTrialContext.from_config(config)
    trial = pytorch_onevar_model.OneVarTrial(context)

    train_ds = trial.build_training_data_loader()
    for epoch_idx in range(3):
        for batch_idx, batch in enumerate(train_ds):
            metrics = trial.train_batch(batch, epoch_idx, batch_idx)
            # Verify the training is correct.
            pytorch_onevar_model.OneVarTrial.check_batch_metrics(metrics, batch_idx)

    eval_ds = trial.build_validation_data_loader()
    for batch in eval_ds:
        metrics = trial.evaluate_batch(batch)


def test_estimator_from_config() -> None:
    config = {"hyperparameters": {"global_batch_size": 4, "learning_rate": 0.001}}
    context = estimator.EstimatorTrialContext.from_config(config)
    trial = estimator_linear_model.LinearEstimator(context)

    eval_spec = trial.build_validation_spec()

    eval_metrics, _ = tf.estimator.train_and_evaluate(
        trial.build_estimator(),
        trial.build_train_spec(),
        tf.estimator.EvalSpec(input_fn=eval_spec.input_fn),
    )
    # Verify the custom reducer and validation datasets are correct.
    assert eval_metrics["label_sum_tensor_fn"] == estimator_linear_model.validation_label_sum()


def test_keras_from_config() -> None:
    data_len = 10
    lr = 0.001
    config = {
        "hyperparameters": {"global_batch_size": 1, "learning_rate": lr, "dataset_range": data_len},
        "searcher": {"metric": "val_loss"},
    }
    context = keras.TFKerasTrialContext.from_config(config)
    trial = tf_keras_one_var_model.OneVarTrial(context)

    model = trial.build_model()
    model.fit(trial.build_training_data_loader(), verbose=0)
    eval_loss = model.evaluate(trial.build_validation_data_loader(), verbose=0)

    # Simulate the training that would happen.
    weight = 0.0
    for _epoch in range(1):
        for data in range(data_len):
            grad = trial.calc_gradient(weight, [data])
            weight -= lr * grad

    # Simluate validation loss.
    sim_loss = trial.calc_loss(weight, range(data_len))

    assert pytest.approx(sim_loss, eval_loss)

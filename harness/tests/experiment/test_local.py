import pathlib
import typing

import pytest
import tensorflow as tf
from _pytest import monkeypatch

import determined as det
from determined import experimental, keras, pytorch
from tests.experiment.fixtures import pytorch_onevar_model


def test_test_one_batch(monkeypatch: monkeypatch.MonkeyPatch, tmp_path: pathlib.Path) -> None:
    def mock_get_tensorboard_path(dummy: typing.Dict[str, typing.Any]) -> pathlib.Path:
        return tmp_path.joinpath("tensorboard")

    monkeypatch.setattr(
        pytorch.PyTorchTrialContext, "get_tensorboard_path", mock_get_tensorboard_path
    )

    with det._local_execution_manager(pathlib.Path(pytorch_onevar_model.__file__).parent):
        experimental.test_one_batch(
            trial_class=pytorch_onevar_model.OneVarTrial,
            config={
                "hyperparameters": {
                    "hidden_size": 2,
                    "learning_rate": 0.5,
                    "global_batch_size": 4,
                    "dataloader_type": "determined",
                },
                "searcher": {"metric": "loss"},
            },
        )


def test_keras_from_config() -> None:
    from tests.experiment.fixtures import tf_keras_one_var_model

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

    # Simulate validation loss.
    sim_loss = trial.calc_loss(weight, range(data_len))

    assert sim_loss == pytest.approx(eval_loss)

import pathlib

from determined import experimental
from tests.unit.experiment.fixtures import pytorch_xor_model


def test_test_one_batch() -> None:
    experimental.test_one_batch(
        pathlib.Path(pytorch_xor_model.__file__).parent,
        pytorch_xor_model.XORTrial,
        config={
            "hyperparameters": {"hidden_size": 2, "learning_rate": 0.5, "global_batch_size": 4}
        },
    )

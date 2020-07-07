import pathlib

import determined as det
from determined import experimental
from tests.experiment.fixtures import pytorch_xor_model


def test_test_one_batch() -> None:
    with det._local_execution_manager(pathlib.Path(pytorch_xor_model.__file__).parent):
        experimental.test_one_batch(
            trial_class=pytorch_xor_model.XORTrial,
            config={
                "hyperparameters": {"hidden_size": 2, "learning_rate": 0.5, "global_batch_size": 4}
            },
        )

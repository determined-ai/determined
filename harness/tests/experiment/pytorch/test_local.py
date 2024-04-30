import pathlib
import typing

from _pytest import monkeypatch

import determined as det
from determined import experimental, pytorch
from tests.experiment import fixtures


def test_test_one_batch(monkeypatch: monkeypatch.MonkeyPatch, tmp_path: pathlib.Path) -> None:
    def mock_get_tensorboard_path(dummy: typing.Dict[str, typing.Any]) -> pathlib.Path:
        return tmp_path.joinpath("tensorboard")

    monkeypatch.setattr(
        pytorch.PyTorchTrialContext, "get_tensorboard_path", mock_get_tensorboard_path
    )

    with det._local_execution_manager(pathlib.Path(fixtures.pytorch_onevar_model.__file__).parent):
        experimental.test_one_batch(
            trial_class=fixtures.pytorch_onevar_model.OneVarTrial,
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

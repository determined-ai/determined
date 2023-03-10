import pathlib
from typing import Any, Dict

from _pytest import monkeypatch

from determined import tensorboard
from determined.tensorboard.metric_writers import pytorch


def test_torch_writer(monkeypatch: monkeypatch.MonkeyPatch, tmp_path: pathlib.Path) -> None:
    def mock_get_base_path(dummy: Dict[str, Any]) -> pathlib.Path:
        return tmp_path

    monkeypatch.setattr(tensorboard, "get_base_path", mock_get_base_path)
    logger = pytorch._TorchWriter()
    logger.add_scalar("foo", 7, 0)
    logger.reset()
    logger.add_scalar("foo", 8, 1)
    logger.reset()

    files = list(tmp_path.iterdir())
    assert len(files) == 2

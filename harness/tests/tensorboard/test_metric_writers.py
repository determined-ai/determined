import pathlib
import time
from typing import Any, Dict
from unittest import mock

from _pytest import monkeypatch
from tensorflow.python.summary import summary_iterator

from determined import tensorboard
from determined.tensorboard.metric_writers import pytorch, tensorflow


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


@mock.patch("determined.tensorboard.get_base_path")
def test_batch_metric_writer(mock_get_base_path: mock.MagicMock, tmp_path: pathlib.Path) -> None:
    """
    This test verifies that writing metrics to Tensorboard quickly in succession effectively
    batches event writes so that a single event file may contain more than one event and
    subsequent writes do not overwrite each other.
    """
    mock_get_base_path.return_value = tmp_path

    writer = tensorflow.TFWriter()
    batch_writer = tensorboard.BatchMetricWriter(writer)

    validation_period = 2

    for i in range(10):
        step = i + 1
        batch_writer.on_train_step_end(steps_completed=i, metrics={"x": step})
        if i % validation_period == 0:
            batch_writer.on_validation_step_end(steps_completed=i, metrics={"x": step})

    # Sleep for 1 second to make subsequent metrics write to a new file.
    time.sleep(1)

    for i in range(10, 20):
        step = i + 1
        batch_writer.on_train_step_end(steps_completed=i, metrics={"x": step})
        if i % validation_period == 0:
            batch_writer.on_validation_step_end(steps_completed=i, metrics={"x": step})

    train_events = []
    val_events = []

    # Read event files saved and verify all metrics are written.
    event_files = list(tmp_path.iterdir())
    for file in event_files:
        for event in summary_iterator.summary_iterator(str(file)):
            # TensorFlow injects an event containing metadata at the start of every tfevent
            # file; ignore these.
            if getattr(event, "file_version", None):
                continue
            for event_data in event.summary.value:
                if event_data.tag == "Determined/x":
                    train_events.append(event_data.simple_value)
                elif event_data.tag == "Determined/val_x":
                    val_events.append(event_data.simple_value)

    assert len(train_events) == 20
    assert len(val_events) == 20 / validation_period

    for i in range(20):
        assert i + 1 in train_events

    for i in range(1, 20, validation_period):
        assert i in val_events

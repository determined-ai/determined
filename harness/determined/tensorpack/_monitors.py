import logging
from typing import Any

from tensorpack import callbacks

from determined import tensorboard


class TFEventWriter(callbacks.TFEventWriter):  # type: ignore
    def __new__(cls, *args: Any, **kwargs: Any) -> callbacks.TFEventWriter:
        fixed_parameters = ["logdir", "split_files"]
        for param in fixed_parameters:
            if param in kwargs:
                logging.warn(f"parameter {param} to TFEventWriter will be ignored")
                kwargs.pop(param)

        # Tensorpacks TFEventWriter requires that the logdir is created before
        # the TFEventWriter is created.
        base_path = tensorboard.get_base_path({})
        base_path.mkdir(parents=True, exist_ok=True)

        kwargs["logdir"] = str(base_path)

        # split_files forces the TFEventWriter to start a new log file after
        # flushing tf events to disk. This creates distinct files that
        # TensorboardManagers expect for syncing tf events to persistent storage.
        kwargs["split_files"] = True

        return callbacks.TFEventWriter(*args, **kwargs)

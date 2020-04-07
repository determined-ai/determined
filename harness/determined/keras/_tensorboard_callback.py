from typing import Any

import tensorflow as tf

from determined import tensorboard


class TFKerasTensorBoard(tf.keras.callbacks.TensorBoard):  # type: ignore
    def __init__(self, *args: Any, **kwargs: Any):
        log_dir = str(tensorboard.get_base_path({}).resolve())
        super().__init__(log_dir=log_dir, *args, **kwargs)

    def _write_logs(self, *args: Any) -> None:
        """
        _write_logs calls the original write logs function from the keras
        TensorBoard callback. After the logs are flushed to disk we close and
        reopen the tf event writer so that it serializes the next set of logs
        to a new file. This allows the tensorboard manager to treat the
        written files as immutable and upload them to persistent storage
        without later having to append to them. This behavior is useful for
        tensorboard backed by S3.
        """
        super()._write_logs(*args)
        self.writer.close()
        self.writer.reopen()
